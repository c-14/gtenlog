package scraper

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"

	s "github.com/c-14/gtenlog/storage"
	_ "github.com/mattn/go-sqlite3"
)

func ScrapeLogs(path string, logs chan s.TenhouLocalStorage, errChan chan error) {
	defer close(logs)

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		errChan <- err
		return
	}
	defer db.Close()

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' LIMIT 1;").Scan(&tableName)
	switch {
	case err == sql.ErrNoRows:
		err = errors.New("Specified SQLite database does not contain any tables")
		fallthrough
	case err != nil:
		errChan <- err
		return
	}

	var rows *sql.Rows
	switch {
	case tableName == "webappsstore2":
		rows, err = db.Query("SELECT value FROM webappsstore2 WHERE scope LIKE 'ten.uohnet%' AND key LIKE 'log%' AND key IS NOT 'lognext';")
	case tableName == "ItemTable":
		rows, err = db.Query("SELECT CAST(value as TEXT) FROM ItemTable WHERE key LIKE 'log%' AND key IS NOT 'lognext';")
	}
	if err != nil {
		errChan <- err
		return
	}
	defer rows.Close()

	for rows.Next() {
		var value s.TenhouLocalStorage
		if err = rows.Scan(&value); err != nil {
			errChan <- err
			return
		}
		logs <- value
	}
	if err = rows.Err(); err != nil {
		errChan <- err
		return
	}
}

func readUserLog(userLogs *map[string]s.UserLogSet, path string, user string) (s.UserLogSet, error) {
	if _, ok := (*userLogs)[user]; !ok {
		var log s.UserLogSet = s.UserLogSet{Path: path, User: user, Logs: make(s.LogSet)}
		err := log.Read()
		if os.IsNotExist(err) {
			(*userLogs)[user] = log
			return log, nil
		} else if err != nil {
			return log, err
		}

		(*userLogs)[user] = log
		return log, nil
	}
	return (*userLogs)[user], nil
}

func WriteLogs(path string, logs chan s.TenhouLocalStorage, errChan chan error, done chan int) {
	var userLogs map[string]s.UserLogSet = make(map[string]s.UserLogSet, 5)

	defer func() {
		for _, log := range userLogs {
			err := log.Write()
			if err != nil {
				errChan <- err
			}
		}
		done <- 1
	}()

	for logItem := range logs {
		logData, err := readUserLog(&userLogs, path, logItem.Users[0])
		if err != nil {
			errChan <- err
			return
		}
		b, err := json.Marshal(logItem)
		if err != nil {
			errChan <- err
			return
		}
		logData.Logs[string(b)] = true
	}
}
