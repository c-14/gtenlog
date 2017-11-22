package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

const mjlogBase string = "http://tenhou.net/3/mjlog2xml.cgi?"
const referBase string = "http://tenhou.net/3/?log="

type logInfo struct {
	log  string
	user string
}

func readLogNames(pathRoot string, logs chan logInfo, errChan chan error) {
	defer close(logs)

	matches, err := filepath.Glob(path.Join(pathRoot, "user", "*", "localLogs.index"))
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

func wrapOpen(fPath string) (*os.File, error) {
	file, err := os.OpenFile(fPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if os.IsNotExist(err) {
		err = os.Mkdir(path.Dir(fPath), 0755)
		if err != nil {
			return nil, err
		}
		return wrapOpen(fPath)
	}
	return file, err
}

func fetchGameLogs(pathRoot string, logs chan logInfo, errChan chan error, done chan int) {
	conn := &http.Client{}

	defer func() { done <- 1 }()
	for log := range logs {
		fPath := path.Join(pathRoot, "user", log.user, "xml", log.log+".xml")
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
