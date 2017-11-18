package main

import (
	"errors"
	"fmt"
	"os"
)

const version = "0.1.0-alpha"

func usage() string {
	return `usage: gtenlog [--help] {scrape|fetch} ...

Subcommands:
	scrape <webappstore.sqlite> <output_path>
	fetch <tenhou_id>`
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
	if len(args) != 1 {
		return errors.New("usage: grue fetch <tenhou_id>")
	}
	// var id string = args[0]
	// TODO: do things
	return nil
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
