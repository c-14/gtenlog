package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

const version = "0.1.0-alpha"

func usage() string {
	return `usage: gtenlog [--help] {scrape|fetch|aggregate} ...

Subcommands:
	scrape <webappstore.sqlite> <output_path>
	fetch <fetchType> <log_root> [-s <date>] [-e <date>]
	aggregate <log_root>`
}

func scrape(args []string) (err error) {
	if len(args) != 2 {
		return errors.New("usage: grue scrape <webappstore.sqlite> <output_path>")
	}
	var db string = args[0]
	var path string = args[1]
	var logs chan TenhouLocalStorage = make(chan TenhouLocalStorage, 10)
	var errors chan error = make(chan error)
	var finished chan int = make(chan int, 1)

	go scrapeLogs(db, logs, errors)
	go writeLogs(path, logs, errors, finished)

	select {
	case err = <-errors:
		return err
	case <-finished:
		return nil
	}
}

func fetch(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: grue fetch <fetchType> <log_root> [-s <date>] [-e <date>]")
	}

	var fetchType string = args[0]
	var path string = args[1]
	var startDate string
	var endDate string

	var fetchFlags = flag.NewFlagSet("fetch", flag.ExitOnError)
	fetchFlags.StringVar(&startDate, "s", getDefaultStartDate(), "First date for which to download daily logs")
	fetchFlags.StringVar(&endDate, "e", getDefaultEndDate(), "Last date for which to download daily logs")
	err := fetchFlags.Parse(args[2:])
	if err != nil {
		return err
	}

	var logs chan logInfo = make(chan logInfo, 10)
	var errChan chan error = make(chan error)
	var finished chan int = make(chan int, 1)

	var done int = 1
	conn := setupHTTP()
	switch {
	case fetchType == "user":
		go readLogNames(path, logs, errChan)
		go fetchGameLogs(conn, path, logs, errChan, finished)
	case fetchType == "daily":
		go fetchSCx(conn, path, startDate, endDate, errChan, finished)
	case fetchType == "yearly":
		go fetchSCRAW(conn, path, errChan, finished)
	case fetchType == "all":
		go readLogNames(path, logs, errChan)
		go fetchGameLogs(conn, path, logs, errChan, finished)
		go fetchSCx(conn, path, startDate, endDate, errChan, finished)
		go fetchSCRAW(conn, path, errChan, finished)
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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage())
		os.Exit(EX_USAGE)
	}
	var err error
	switch cmd := os.Args[1]; cmd {
	case "scrape":
		err = scrape(os.Args[2:])
	case "fetch":
		err = fetch(os.Args[2:])
	case "aggregate":
		err = aggregate(os.Args[2:])
	case "-v":
		fallthrough
	case "--version":
		fmt.Println(version)
	case "-h":
		fallthrough
	case "--help":
		fmt.Println(usage())
	default:
		fmt.Fprintln(os.Stderr, usage())
		os.Exit(EX_USAGE)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
