package cmd

import (
	"errors"
	"flag"
	"github.com/c-14/gtenlog/tenhou"
	"github.com/c-14/gtenlog/storage"
)

func Fetch(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: grue fetch <fetchType> <log_root> [-s <date>] [-e <date>]")
	}

	var fetchType string = args[0]
	var path string = args[1]
	var startDate, endDate string

	var fetchFlags = flag.NewFlagSet("fetch", flag.ExitOnError)
	fetchFlags.StringVar(&startDate, "s", getDefaultStartDate(), "First date for which to download daily logs")
	fetchFlags.StringVar(&endDate, "e", getDefaultEndDate(), "Last date for which to download daily logs")
	err := fetchFlags.Parse(args[2:])
	if err != nil {
		return err
	}

	var logs chan storage.UserLogInfo = make(chan storage.UserLogInfo, 10)
	var errChan chan error = make(chan error)
	var finished chan int = make(chan int, 1)

	archive := storage.LogArchive{PathRoot: path}

	var done int = 1
	conn := tenhou.SetupHTTP()
	switch {
	case fetchType == "user":
		go archive.GetUserLogNames("*", logs, errChan)
		go tenhou.FetchGameLogs(conn, archive, logs, errChan, finished)
	case fetchType == "daily":
		go tenhou.FetchSCx(conn, archive, startDate, endDate, errChan, finished)
	case fetchType == "yearly":
		go tenhou.FetchSCRAW(conn, archive, errChan, finished)
	case fetchType == "all":
		go archive.GetUserLogNames("*", logs, errChan)
		go tenhou.FetchGameLogs(conn, archive, logs, errChan, finished)
		go tenhou.FetchSCx(conn, archive, startDate, endDate, errChan, finished)
		go tenhou.FetchSCRAW(conn, archive, errChan, finished)
		done = 3
	default:
		return errors.New("fetchType must be one of [user, daily, yearly, all]")
	}

	for {
		if done == 0 {
			return nil
		}
		select {
		case err := <-errChan:
			// fmt.Fprintln(os.Stderr, err)
			return err
		case <-finished:
			done--
		}
	}
}

