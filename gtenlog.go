package main

import (
	"fmt"
	"os"

	"github.com/c-14/gtenlog/cmd"
)

const version = "0.1.0-beta"

func usage() string {
	return `usage: gtenlog [--help] {scrape|fetch|aggregate} ...

Subcommands:
	scrape <webappstore.sqlite> <output_path>
	fetch <fetchType> <log_root> [-s <date>] [-e <date>]
	aggregate <log_root>
	users <userFile> {add|addAlias|list}
	`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage())
		os.Exit(EX_USAGE)
	}
	var err error
	switch command := os.Args[1]; command {
	case "scrape":
		err = cmd.Scrape(os.Args[2:])
	case "fetch":
		err = cmd.Fetch(os.Args[2:])
	case "aggregate":
		err = cmd.Aggregate(os.Args[2:])
	case "grep":
		err = cmd.Grep(os.Args[2:])
	case "users":
		err = cmd.Users(os.Args[2:])
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
