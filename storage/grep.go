package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type walkFileError struct {
	path string
	err error
}

func (e walkFileError) Error() string {
	return fmt.Sprintf("Failed to parse %s: %v", e.path, e.err.Error())
}

func (e walkFileError) IsNotExist() bool {
	return os.IsNotExist(e.err)
}

func (a LogArchive) GrepLogs(lobby string, aliases UserListing, startDate time.Time, endDate time.Time, logs chan SCxLogLine, errChan chan error, done chan int) {
	defer func() { done <- 1 }()

	if !(lobby[0] == 'L' || lobby[0] == 'l') || len(lobby) != 5 {
		errChan <- fmt.Errorf("Invalid Lobby Format, expecting L[0-9]{4}, got %s", lobby)
		return
	}

	var err error
	for y := startDate.Year(); y <= endDate.Year(); y++ {
		err = filepath.Walk(filepath.Join(a.PathRoot, "sca", strconv.Itoa(y)), 
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return walkFileError{path, err}
			}
			if info.IsDir() {
				return nil
			}
			if !info.Mode().IsRegular() {
				return walkFileError{path, errors.New("not a regular file")}
			}

			scxLog, err := InitSCxLogParser(path)
			if err != nil {
				return walkFileError{path, err}
			}
			defer scxLog.Close()

			if scxLog.Date.Before(startDate) || scxLog.Date.After(endDate) {
				return nil
			}

			for scxLog.Scan() {
				switch v := scxLog.Token().(type) {
				case *SCALogLine:
					if v.Lobby != lobby {
						continue
					}
					match := false
					for i, score := range(v.Score) {
						userName, ok := aliases.AliasMap[score.UserName]
						if !ok {
							userName = score.UserName
						} else {
							v.Score[i].UserName = userName
						}
						if _, ok := aliases.Users[userName]; ok {
							match = true
						}
					}
					if !match {
						continue
					}
					logs <- v
				default:
					return walkFileError{path, errors.New("Log Type not yet implemented")}
				}
			}
			if err = scxLog.Err(); err != nil {
				return walkFileError{path, err}
			}

			return nil
		})
		if err != nil {
			if err.(walkFileError).IsNotExist() {
				errChan <- fmt.Errorf("No data for year %v, aborting", strconv.Itoa(y))
			}
			break
		}
	}
	if err != nil {
		errChan <- err
	}
	return
}
