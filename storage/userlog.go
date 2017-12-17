package storage

import (
	"path/filepath"
	"os"
)

type UserLog struct {
	file *os.File
}

type UserLogInfo struct {
	LogID  string
	User string
}

func (ul UserLog) Close() error {
	return ul.file.Close()
}

func (ul UserLog) Write(p []byte) (int, error) {
	return ul.file.Write(p)
}

func (a LogArchive) AddUserLog(info UserLogInfo) (UserLog, error) {
	var log UserLog
	var err error

	fPath := filepath.Join(a.PathRoot, "user", info.User, "xml", info.LogID+".xml")
	log.file, err = wrapOpen(fPath)

	return log, err
}
