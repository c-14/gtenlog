package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LogArchive struct {
	PathRoot string
}

type SCRAWLogInfo struct {
	statInfo  os.FileInfo
	existsErr error
	file      *os.File
	path      string
}

type SCxLogInfo struct {
	statInfo  os.FileInfo
	existsErr error
	file      *os.File
	path      string
}

type LogInfo interface {
	Exists() (bool, error)
	IsComplete(int64) bool

	Open() error
	Close() error
	Write([]byte) (int, error)
	Remove() error
}

func (a LogArchive) GetUserLogNames(user string, logs chan UserLogInfo, errChan chan error) {
	defer close(logs)

	matches, err := filepath.Glob(filepath.Join(a.PathRoot, "user", user, "localLogs.index"))
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
			logs <- UserLogInfo{LogID: tls.Log, User: tls.Users[0]}
		}
		if err = lines.Err(); err != nil {
			errChan <- err
			return
		}
	}
}

func (l SCxLogInfo) Exists() (bool, error) {
	if l.existsErr == nil && l.statInfo.Mode().IsRegular() {
		return true, nil
	} else if l.existsErr == nil {
		return false, fmt.Errorf("%s not a regular file, aborting", l.path)
	} else if l.existsErr != nil && !os.IsNotExist(l.existsErr) {
		return false, l.existsErr
	}
	// File doesn't exist
	return false, nil
}

func (l SCxLogInfo) IsComplete(rLength int64) bool {
	fLength := l.statInfo.Size()
	if fLength >= rLength {
		return true
	}
	return false
}

func (l SCxLogInfo) Remove() error {
	return os.Remove(l.path)
}

func (l *SCxLogInfo) Close() error {
	return l.file.Close()
}

func (l *SCxLogInfo) Write(p []byte) (int, error) {
	return l.file.Write(p)
}

func (l *SCxLogInfo) Open() (err error) {
	l.file, err = wrapOpen(l.path)
	return
}

func (a LogArchive) AddSCxLogInfo(scx string, date time.Time, fName string) SCxLogInfo {
	var log SCxLogInfo

	log.path = filepath.Join(a.PathRoot, scx, date.Format("2006"), date.Format("01"), fName)
	log.statInfo, log.existsErr = os.Stat(log.path)

	return log
}

func (l SCRAWLogInfo) Exists() (bool, error) {
	if l.existsErr == nil && l.statInfo.Mode().IsRegular() {
		return true, nil
	} else if l.existsErr == nil {
		return false, fmt.Errorf("%s not a regular file, aborting", l.path)
	} else if l.existsErr != nil && !os.IsNotExist(l.existsErr) {
		return false, l.existsErr
	}
	// File doesn't exist
	return false, nil
}

func (l SCRAWLogInfo) IsComplete(rLength int64) bool {
	fLength := l.statInfo.Size()
	if fLength == rLength {
		return true
	}
	return false
}

func (l SCRAWLogInfo) Remove() error {
	return os.Remove(l.path)
}

func (l *SCRAWLogInfo) Close() error {
	return l.file.Close()
}

func (l *SCRAWLogInfo) Write(p []byte) (int, error) {
	return l.file.Write(p)
}

func (l *SCRAWLogInfo) Open() (err error) {
	l.file, err = wrapOpen(l.path)
	return
}

func (a LogArchive) AddSCRAWLogInfo(year int) SCRAWLogInfo {
	var log SCRAWLogInfo

	log.path = filepath.Join(a.PathRoot, "scraw", fmt.Sprintf("scraw%d.zip", year))
	log.statInfo, log.existsErr = os.Stat(log.path)

	return log
}
