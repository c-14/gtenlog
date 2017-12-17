package tenhou

import (
	"bufio"
	"fmt"
	s "github.com/c-14/gtenlog/storage"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

const mjlogBase string = "http://tenhou.net/3/mjlog2xml.cgi?"
const referBase string = "http://tenhou.net/3/?log="
const scrawBase string = "http://tenhou.net/sc/raw"

func SetupHTTP() *http.Client {
	return &http.Client{}
}

func FetchGameLogs(conn *http.Client, archive s.LogArchive, logs chan s.UserLogInfo, errChan chan error, done chan int) {
	defer func() { done <- 1 }()
	for log := range logs {
		ul, err := archive.AddUserLog(log)
		if os.IsExist(err) {
			continue
		}
		defer ul.Close()

		req, err := http.NewRequest("GET", mjlogBase+log.LogID, nil)
		if err != nil {
			errChan <- err
			return
		}
		req.Header.Add("Referer", referBase+log.LogID)
		resp, err := conn.Do(req)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errChan <- fmt.Errorf("GET request for %s failed: %s", mjlogBase+log.LogID, http.StatusText(resp.StatusCode))
			return
		}

		wrLog := bufio.NewWriter(ul)
		_, err = wrLog.ReadFrom(resp.Body)
		if err != nil {
			errChan <- err
			return
		}
		err = wrLog.Flush()
		if err != nil {
			errChan <- err
			return
		}
	}
}

func fetchArchivedLog(conn *http.Client, logInfo s.LogInfo, logURL string) error {
	exists, err := logInfo.Exists()
	if exists {
		// File exists, check that length matches remote
		resp, err := conn.Head(logURL)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HEAD request for %s failed: %s", logURL, http.StatusText(resp.StatusCode))
		}
		rLength := resp.ContentLength

		if logInfo.IsComplete(rLength) {
			return nil
		}

		err = logInfo.Remove()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// File does not exist yet, so download
	resp, err := conn.Get(logURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET request for %s failed: %s", logURL, http.StatusText(resp.StatusCode))
	}

	err = logInfo.Open()
	if err != nil {
		return err
	}
	defer logInfo.Close()

	wrLog := bufio.NewWriter(logInfo)
	_, err = wrLog.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	return wrLog.Flush()
}

func FetchSCRAW(conn *http.Client, archive s.LogArchive, errChan chan error, done chan int) {
	defer func() { done <- 1 }()

	japan, _ := time.LoadLocation("Japan")
	currentYear := time.Now().In(japan).Year()
	for year := 2006; year < currentYear; year++ {
		logURL, _ := url.Parse(scrawBase)
		logURL.Path = path.Join(logURL.Path, fmt.Sprintf("scraw%d.zip", year))
		logInfo := archive.AddSCRAWLogInfo(year)

		err := fetchArchivedLog(conn, &logInfo, logURL.String())
		if err != nil {
			errChan <- err
			return
		}
	}
}

func getLogList(conn *http.Client, old bool, logList *io.ReadCloser) error {
	listURL, _ := url.Parse(scrawBase)
	listURL.Path = path.Join(listURL.Path, "list.cgi")
	if old {
		listURL.RawQuery = "old"
	}

	resp, err := conn.Get(listURL.String())
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("GET request for %s failed: %s", listURL, http.StatusText(resp.StatusCode))
	}

	*logList = resp.Body
	return nil
}

func fetchSCxLogs(conn *http.Client, archive s.LogArchive, startDate time.Time, endDate time.Time, japan *time.Location, old bool) error {
	var logList io.ReadCloser
	err := getLogList(conn, old, &logList)
	if err != nil {
		return err
	}
	defer logList.Close()

	parser, err := InitLogListParser(logList)
	if err != nil {
		return err
	}
	for parser.Scan() {
		var scx string
		var date time.Time

		tok := parser.Token()
		if old {
			scx = tok.File[5:8]
			date, err = time.ParseInLocation("20060102", tok.File[8:16], japan)
		} else {
			scx = tok.File[:3]
			date, err = time.ParseInLocation("20060102", tok.File[3:11], japan)
		}
		if err != nil {
			return err
		}

		if date.Before(startDate) || date.After(endDate) {
			continue
		}

		logInfo := archive.AddSCxLogInfo(scx, date, path.Base(tok.File))

		if exists, err := logInfo.Exists(); err != nil {
			return err
		} else if exists {
			continue
		}

		logURL, _ := url.Parse(scrawBase)
		logURL.Path = path.Join(logURL.Path, "dat", tok.File)

		err = fetchArchivedLog(conn, &logInfo, logURL.String())
		if err != nil {
			return err
		}
	}
	if err = parser.Err(); err != nil {
		return err
	}
	return nil
}

func FetchSCx(conn *http.Client, archive s.LogArchive, startDate string, endDate string, errChan chan error, done chan int) {
	defer func() { done <- 1 }()

	japan, err := time.LoadLocation("Japan")
	if err != nil {
		errChan <- err
		return
	}
	start, err := time.ParseInLocation("2006-01-02", startDate, japan)
	if err != nil {
		errChan <- fmt.Errorf("Failed to parse startDate: %s", err)
		return
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, japan)
	if err != nil {
		errChan <- fmt.Errorf("Failed to parse endDate: %s", err)
		return
	}

	// TODO: check what happens when the year rolls over
	now := time.Now().In(japan)
	now = time.Date(now.Year(), now.Month(), now.Day(), 00, 00, 00, 00, japan)
	if start.Year() < now.Year() {
		// TODO: need to unpack/fetch scraw files here
		start = time.Date(now.Year(), 01, 01, 00, 00, 00, 00, japan)
	}

	cutoff := now.AddDate(0, 0, -8)
	if start.Before(cutoff) {
		err = archive.AggregateLogs(japan, cutoff)
		err = fetchSCxLogs(conn, archive, start, end, japan, true)
	}
	if err != nil {
		errChan <- err
		return
	}
	if end.After(cutoff) {
		err = fetchSCxLogs(conn, archive, start, end, japan, false)
	}
	if err != nil {
		errChan <- err
		return
	}
}
