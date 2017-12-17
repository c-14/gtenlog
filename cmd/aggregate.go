package cmd

import (
	"errors"
	"github.com/c-14/gtenlog/storage"
	"time"
)

func Aggregate(args []string) (err error) {
	if len(args) != 1 {
		return errors.New("usage: grue aggregate <log_root>")
	}
	var path string = args[0]

	japan, _ := time.LoadLocation("Japan")
	now := time.Now().In(japan)
	now = time.Date(now.Year(), now.Month(), now.Day(), 00, 00, 00, 00, japan)

	archive := storage.LogArchive{PathRoot: path}
	return archive.AggregateLogs(japan, now.AddDate(0, 0, -8))
}
