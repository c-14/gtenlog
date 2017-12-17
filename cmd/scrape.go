package cmd

import (
	"errors"
	"github.com/c-14/gtenlog/scraper"
	"github.com/c-14/gtenlog/storage"
)

func Scrape(args []string) (err error) {
	if len(args) != 2 {
		return errors.New("usage: grue scrape <webappstore.sqlite> <output_path>")
	}
	var db string = args[0]
	var path string = args[1]
	var logs chan storage.TenhouLocalStorage = make(chan storage.TenhouLocalStorage, 10)
	var errors chan error = make(chan error)
	var finished chan int = make(chan int, 1)

	go scraper.ScrapeLogs(db, logs, errors)
	go scraper.WriteLogs(path, logs, errors, finished)

	select {
	case err = <-errors:
		return err
	case <-finished:
		return nil
	}
}

