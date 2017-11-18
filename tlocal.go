package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path"
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

func openUserLog(userLogs *map[string]*os.File, pathname string, user string) (*os.File, error) {
	if _, ok := (*userLogs)[user]; !ok {
		fPath := path.Join(pathname, user, "localLogs.index")
		os.MkdirAll(path.Dir(fPath), 0755)
		file, err := os.OpenFile(fPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		(*userLogs)[user] = file
		return file, nil
	}
	return (*userLogs)[user], nil
}

func writeLogs(path string, logs chan TenhouLocalStorage, errors chan error, done chan int) {
	var userLogs map[string]*os.File = make(map[string]*os.File, 5)

	defer func() {
		for _, file := range userLogs {
			file.Close()
		}
	}()

	for logItem := range logs {
		file, err := openUserLog(&userLogs, path, logItem.Users[0])
		if err != nil {
			errors <- err
			return
		}
		b, err := json.Marshal(logItem)
		if err != nil {
			errors <- err
			return
		}
		file.Write(b)
		file.WriteString("\n")
	}

	done <- 1
}
