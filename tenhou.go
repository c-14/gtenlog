package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"
)

const mjlogBase string = "http://tenhou.net/3/mjlog2xml.cgi?"
const referBase string = "http://tenhou.net/3/?log="
const scrawBase string = "http://tenhou.net/sc/raw"

type logInfo struct {
	log  string
	user string
}

func setupHTTP() *http.Client {
	return &http.Client{}
}

func readLogNames(pathRoot string, logs chan logInfo, errChan chan error) {
	defer close(logs)

	matches, err := filepath.Glob(filepath.Join(pathRoot, "user", "*", "localLogs.index"))
	if err != nil {
		errChan <- err
		return
	}

	for _, logFile := range matches {
		file, err := os.Open(logFile)
		if err != nil {
			errChan <- err
			return
		}
		defer file.Close()

		lines := bufio.NewScanner(file)
		for lines.Scan() {
			var tls TenhouLocalStorage
			err = json.Unmarshal(lines.Bytes(), &tls)
			if err != nil {
				errChan <- err
				return
			}
			logs <- logInfo{log: tls.Log, user: tls.Users[0]}
		}
		if err = lines.Err(); err != nil {
			errChan <- err
			return
		}
	}
}

// func writeGameLog(file *os.File, body io.ReadCloser, errChan chan error, done chan int) {
// 	defer done <- 1
// 	defer file.Close()
// 	defer body.Close()

// 	wrLog := bufio.NewWriter(file)
// 	_, err := wrLog.ReadFrom(body)
// 	if err != nil {
// 		errChan <- err
//		return
// 	}
// 	err = wrLog.Flush()
// 	if err != nil {
// 		errChan <- err
//		return
// 	}
// }

func wrapOpen(fPath string) (*os.File, error) {
	file, err := os.OpenFile(fPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if os.IsNotExist(err) {
		err = os.Mkdir(filepath.Dir(fPath), 0755)
		if err != nil {
			return nil, err
		}
		return wrapOpen(fPath)
	}
	return file, err
}

func fetchGameLogs(conn *http.Client, pathRoot string, logs chan logInfo, errChan chan error, done chan int) {
	// var ioErr chan error = make(chan error)
	// var ioWait chan int = make(chan int)

	// var io_cnt int = 0
	// defer func {
	// 	for i := 0; i < io_cnt; i++ {
	// 		<- ioWait
	// 	}
	// 	for err := range ioErr {
	// 		errChan <- err
	// 	}
	// 	done <- 1
	// }()

	defer func() { done <- 1 }()
	for log := range logs {
		fPath := filepath.Join(pathRoot, "user", log.user, "xml", log.log+".xml")
		file, err := wrapOpen(fPath)
		if os.IsExist(err) {
			continue
		}
		defer file.Close()

		req, err := http.NewRequest("GET", mjlogBase+log.log, nil)
		if err != nil {
			errChan <- err
			return
		}
		req.Header.Add("Referer", referBase+log.log)
		resp, err := conn.Do(req)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errChan <- fmt.Errorf("GET request for %s failed: %s", mjlogBase+log.log, http.StatusText(resp.StatusCode))
			return
		}

		// io_cnt += 1
		// go writeGameLog(file, resp.Body, ioErr, ioWait)
		wrLog := bufio.NewWriter(file)
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

func getDefaultStartDate() string {
	japan, _ := time.LoadLocation("Japan")
	return time.Now().In(japan).AddDate(0, 0, -8).Format("2006-01-02")
}

func getDefaultEndDate() string {
	japan, _ := time.LoadLocation("Japan")
	return time.Now().In(japan).Format("2006-01-02")
}

func fetchArchivedLog(conn *http.Client, logPath string, logURL string) error {
	logInfo, err := os.Stat(logPath)
	if err == nil && logInfo.Mode().IsRegular() {
		// File exists, check that length matches remote
		resp, err := conn.Head(logURL)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HEAD request for %s failed: %s", logURL, http.StatusText(resp.StatusCode))
		}
		rLength := resp.ContentLength
		fLength := logInfo.Size()
		if fLength == rLength {
			return nil
		}
		os.Remove(logPath)
	} else if err == nil {
		return fmt.Errorf("%s not a regular file, aborting.", logPath)
	} else if err != nil && !os.IsNotExist(err) {
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

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	wrLog := bufio.NewWriter(file)
	_, err = wrLog.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	return wrLog.Flush()
}

func fetchSCRAW(conn *http.Client, pathRoot string, errChan chan error, done chan int) {
	defer func() { done <- 1 }()

	if err := os.MkdirAll(filepath.Join(pathRoot, "scraw"), 0755); err != nil && !os.IsExist(err) {
		errChan <- err
		return
	}

	japan, _ := time.LoadLocation("Japan")
	currentYear := time.Now().In(japan).Year()
	for year := 2006; year < currentYear; year++ {
		logURL, _ := url.Parse(scrawBase)
		logURL.Path = path.Join(logURL.Path, fmt.Sprintf("scraw%d.zip", year))
		logPath := filepath.Join(pathRoot, "scraw", fmt.Sprintf("scraw%d.zip", year))

		err := fetchArchivedLog(conn, logPath, logURL.String())
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

func checkExists(logPath string, rLength int64) (bool, error) {
	logInfo, err := os.Stat(logPath)
	if err == nil && logInfo.Mode().IsRegular() {
		// File exists, check that length matches remote
		fLength := logInfo.Size()
		if fLength == rLength {
			return true, nil
		}
		os.Remove(logPath)
	} else if err == nil {
		return false, fmt.Errorf("%s not a regular file, aborting.", logPath)
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	// File doesn't exist
	return false, nil
}

func fetchSCxLogs(conn *http.Client, pathRoot string, startDate time.Time, endDate time.Time, japan *time.Location, old bool) error {
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

		err = os.MkdirAll(filepath.Join(pathRoot, scx, date.Format("2006"), date.Format("01")), 0755)
		if err != nil && !os.IsExist(err) {
			return err
		}

		fName := path.Base(tok.File)
		logPath := filepath.Join(pathRoot, scx, date.Format("2006"), date.Format("01"), fName)
		logURL, _ := url.Parse(scrawBase)
		logURL.Path = path.Join(logURL.Path, "dat", tok.File)

		if exists, err := checkExists(logPath, tok.Size); err != nil {
			return err
		} else if exists {
			continue
		}

		err = fetchArchivedLog(conn, logPath, logURL.String())
		if err != nil {
			return err
		}
	}
	if err = parser.Err(); err != nil {
		return err
	}
	return nil
}

func fetchSCx(conn *http.Client, pathRoot string, startDate string, endDate string, errChan chan error, done chan int) {
	defer func() { done <- 1 }()

	japan, err := time.LoadLocation("Japan")
	if err != nil {
		errChan <- err
		return
	}
	start, err := time.ParseInLocation("2006-01-02", startDate, japan)
	if err != nil {
		errChan <- err
		return
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, japan)
	if err != nil {
		errChan <- err
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
		err = aggregateLogs(pathRoot, japan, cutoff)
		err = fetchSCxLogs(conn, pathRoot, start, end, japan, true)
	}
	if err != nil {
		errChan <- err
		return
	}
	if end.After(cutoff) {
		err = fetchSCxLogs(conn, pathRoot, start, end, japan, false)
	}
	if err != nil {
		errChan <- err
		return
	}
}
