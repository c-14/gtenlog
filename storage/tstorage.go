package storage

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type TenhouLocalStorage struct {
	Type     int      `json:"type"`
	Lobby    int      `json:"lobby"`
	Log      string   `json:"log"`
	Position int      `json:"oya"`
	Users    []string `json:"uname"`
	Sc       string   `json:"sc"`
}

func (t *TenhouLocalStorage) Scan(src interface{}) error {
	value, ok := src.([]byte)
	if !ok {
		return errors.New("Incorrect SQLite database or code out of date")
	}

	return json.Unmarshal(value, t)
}

type Matches struct {
	matches []string
	date    time.Time

	length int
	low    int
	high   int

	err error
}

func (m Matches) Err() error {
	return m.err
}

func (m *Matches) FindNextSlice(cutoff time.Time, japan *time.Location) bool {
	m.low = m.high

	if m.low >= m.length {
		return false
	}

	var err error
	m.date, err = time.ParseInLocation("20060102", filepath.Base(m.matches[m.low])[3:11], japan)
	if err != nil {
		m.err = err
		return false
	} else if !m.date.Before(cutoff) {
		return false
	}

	for i, possibleMatch := range m.matches[m.low:] {
		pDate, err := time.ParseInLocation("20060102", filepath.Base(possibleMatch)[3:11], japan)
		if err != nil {
			m.err = err
			return false
		}
		if !pDate.Equal(m.date) {
			m.high = i + m.low
			return true
		}
	}
	m.high = m.length
	return true
}

func (m Matches) GetSlice() ([]string, time.Time) {
	return m.matches[m.low:m.high], m.date
}

func (m Matches) Length() int {
	return m.length
}

func GetMatches(pathRoot string, scx string) (m Matches, err error) {
	m.matches, err = filepath.Glob(filepath.Join(pathRoot, scx, "*", "*", "sc???????????.*.gz"))
	if err == nil {
		m.length = len(m.matches)
		sort.Strings(m.matches)
	}
	return
}

func isComplete(logPath string, partial []string) (bool, error) {
	cInfo, err := os.Stat(logPath)
	if err != nil {
		return false, err
	} else if !cInfo.Mode().IsRegular() {
		return false, fmt.Errorf("%s not a regular file, aborting", logPath)
	}

	var tSize int64 = 0
	for _, partialLog := range partial {
		pInfo, err := os.Stat(partialLog)
		if err != nil {
			return false, err
		} else if !pInfo.Mode().IsRegular() {
			return false, fmt.Errorf("%s not a regular file, aborting", partialLog)
		}
		tSize += pInfo.Size()
	}
	cSize := cInfo.Size()
	if tSize * 95 / 100 <= cSize && cSize <= tSize * 105 / 100 {
		return true, nil
	}
	return false, nil
}

func aggregateSlice(pathRoot, scx string, slice []string, date time.Time) error {
	var fName string
	if strings.Compare(scx, "scc") == 0 {
		fName = fmt.Sprintf("scc%s.html.gz", date.Format("20060102"))
	} else {
		fName = fmt.Sprintf("%s%s.log.gz", scx, date.Format("20060102"))
	}
	logPath := filepath.Join(pathRoot, scx, date.Format("2006"), date.Format("01"), fName)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if os.IsExist(err) {
		var complete bool
		complete, err = isComplete(logPath, slice)
		if err != nil {
			return err
		} else if !complete {
			err = os.Remove(logPath)
			if err == nil {
				file, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
			}
		} else {
			for _, partialLog := range slice {
				os.Remove(partialLog)
			}
			return nil
		}
	}
	if err != nil {
		return err
	}
	defer file.Close()
	gzLog, _ := gzip.NewWriterLevel(file, gzip.BestCompression)
	wrLog := bufio.NewWriter(gzLog)

	for _, partialLog := range slice {
		logFile, err := os.Open(partialLog)
		if err != nil {
			return err
		}
		defer logFile.Close()

		reader, err := gzip.NewReader(logFile)
		if err != nil {
			return err
		}
		_, err = wrLog.ReadFrom(reader)
		if err != nil {
			return err
		}
		err = reader.Close()
		if err != nil {
			return err
		}
		os.Remove(partialLog)
	}
	if err = wrLog.Flush(); err != nil {
		return err
	}
	if err = gzLog.Close(); err != nil {
		return err
	}
	return nil
}

func (a LogArchive) AggregateLogs(japan *time.Location, cutoff time.Time) error {
	for _, scx := range []string{"scb", "scc", "scd", "sce"} {
		matches, err := GetMatches(a.PathRoot, scx)
		if err != nil {
			return err
		}
		if matches.Length() == 0 {
			continue
		}

		for matches.FindNextSlice(cutoff, japan) {
			slice, date := matches.GetSlice()

			err = aggregateSlice(a.PathRoot, scx, slice, date)
			if err != nil {
				return err
			}
		}
		if err = matches.Err(); err != nil {
			return err
		}
	}
	return nil
}
