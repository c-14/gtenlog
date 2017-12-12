package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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

	var err error
	m.date, err = time.ParseInLocation("20060102", filepath.Base(m.matches[m.low])[3:11], japan)
	if err != nil {
		m.err = err
		return false
	} else if m.date.After(cutoff) {
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

func aggregateLogs(pathRoot string, japan *time.Location, cutoff time.Time) error {
	for _, scx := range []string{"scb", "scc", "scd", "sce"} {
		matches, err := GetMatches(pathRoot, scx)
		if err != nil {
			return err
		}
		if matches.Length() == 0 {
			continue
		}

		for matches.FindNextSlice(cutoff, japan) {
			slice, date := matches.GetSlice()

			var fName string
			if strings.Compare(scx, "scc") == 0 {
				fName = fmt.Sprintf("scc%s.html.gz", date.Format("20060102"))
			} else {
				fName = fmt.Sprintf("%s%s.log.gz", scx, date.Format("20060102"))
			}
			logPath := filepath.Join(pathRoot, scx, date.Format("2006"), date.Format("01"), fName)

			file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
			if err != nil {
				return err
			}
			defer file.Close()
			wrLog := bufio.NewWriter(file)

			for _, partialLog := range slice {
				logFile, err := os.Open(partialLog)
				if err != nil {
					return err
				}
				defer logFile.Close()

				_, err = wrLog.ReadFrom(logFile)
				if err != nil {
					return err
				}
				defer os.Remove(partialLog)
			}
			err = wrLog.Flush()
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

func aggregate(args []string) (err error) {
	if len(args) != 1 {
		return errors.New("usage: grue aggregate <log_root>")
	}
	var path string = args[0]

	japan, _ := time.LoadLocation("Japan")
	now := time.Now().In(japan)
	now = time.Date(now.Year(), now.Month(), now.Day(), 00, 00, 00, 00, japan)

	return aggregateLogs(path, japan, now.AddDate(0, 0, -8))
}
