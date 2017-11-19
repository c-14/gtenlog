package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path"
)

type LogSet map[string]bool

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

func scrapeLogs(path string, logs chan TenhouLocalStorage, errors chan error) {
	defer close(logs)

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		errors <- err
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT value FROM webappsstore2 WHERE scope LIKE 'ten.uohnet%' AND key LIKE 'log%' AND key IS NOT 'lognext';")
	if err != nil {
		errors <- err
		return
	}
	defer rows.Close()

	for rows.Next() {
		var value TenhouLocalStorage
		if err = rows.Scan(&value); err != nil {
			errors <- err
			return
		}
		logs <- value
	}
	if err = rows.Err(); err != nil {
		errors <- err
		return
	}
}

func writeUserLog(logData LogSet, pathname string, user string) error {
	fPath := path.Join(pathname, user, "localLogs.index")
	file, err := os.OpenFile(fPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if os.IsNotExist(err) {
		os.MkdirAll(path.Dir(fPath), 0755)
		return writeUserLog(logData, pathname, user)
	} else if err != nil {
		return err
	}
	defer file.Close()

	wrLines := bufio.NewWriter(file)
	for data, _ := range logData {
		_, err = wrLines.WriteString(data)
		if err != nil {
			return err
		}
		err = wrLines.WriteByte(0x0A)
		if err != nil {
			return err
		}
	}
	return wrLines.Flush()
}

func readUserLog(userLogs *map[string]LogSet, pathname string, user string) (LogSet, error) {
	if _, ok := (*userLogs)[user]; !ok {
		var logs LogSet
		fPath := path.Join(pathname, user, "localLogs.index")
		file, err := os.Open(fPath)
		if os.IsNotExist(err) {
			logs = make(LogSet)
			(*userLogs)[user] = logs
			return logs, nil
		} else if err != nil {
			return logs, err
		}
		defer file.Close()

		logs = make(LogSet)
		lines := bufio.NewScanner(file)
		for lines.Scan() {
			s := lines.Text()
			logs[s] = true
		}
		if err = lines.Err(); err != nil {
			return logs, err
		}
		(*userLogs)[user] = logs

		return logs, nil
	}
	return (*userLogs)[user], nil
}

func writeLogs(path string, logs chan TenhouLocalStorage, errors chan error, done chan int) {
	var userLogs map[string]LogSet = make(map[string]LogSet, 5)

	defer func() {
		for user, logData := range userLogs {
			err := writeUserLog(logData, path, user)
			if err != nil {
				errors <- err
			}
		}
		done <- 1
	}()

	for logItem := range logs {
		logData, err := readUserLog(&userLogs, path, logItem.Users[0])
		if err != nil {
			errors <- err
			return
		}
		b, err := json.Marshal(logItem)
		if err != nil {
			errors <- err
			return
		}
		logData[string(b)] = true
	}
}
