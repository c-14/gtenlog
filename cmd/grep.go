package cmd

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"encoding/json"

	"github.com/c-14/gtenlog/storage"
)

var grepUsage error = errors.New("usage: gtenlog grep [-s <date>] [-e <date>] [-a <userFile>] <lobby> <logRoot>")

func outputLogLine(oFormat string, log storage.SCxLogLine) error {
	switch {
	case oFormat == "tenhou":
		fmt.Println(log)
		return nil
	case oFormat == "json":
		j, err := json.Marshal(log)
		fmt.Printf("%s", string(j))
		return err
	case oFormat == "jsonlines":
		j, err := json.Marshal(log)
		fmt.Println(string(j))
		return err
	default:
		return fmt.Errorf("No such output format, %s", oFormat)
	}
}

func Grep(args []string) error {
	if len(args) < 2 {
		return grepUsage
	}
	var lobby string
	var startDate, endDate string
	var userPath string
	var oFormat string

	var grepFlags = flag.NewFlagSet("grep", flag.ExitOnError)
	grepFlags.StringVar(&startDate, "s", "2006-07-01", "First date for which to output data")
	grepFlags.StringVar(&endDate, "e", getDefaultEndDate(), "Last date for which to output data")
	grepFlags.StringVar(&userPath, "a", "", "Path to json file containing user/alias mapping")
	grepFlags.StringVar(&oFormat, "f", "tenhou", "Format used to output results [tenhou/json/jsonlines]")
	err := grepFlags.Parse(args)
	if err != nil {
		return err
	}

	if grepFlags.NArg() != 2 {
		return grepUsage
	}
	lobby = grepFlags.Arg(0)
	archive := storage.LogArchive{PathRoot: grepFlags.Arg(1)}

	users, err := storage.ParseUserFile(userPath)
	if err != nil {
		return fmt.Errorf("Error parsing user mapping: %s", err)
	}

	japan, _ := time.LoadLocation("Japan")
	start, err := time.ParseInLocation("2006-01-02", startDate, japan)
	if err != nil {
		return fmt.Errorf("Failed to parse startDate: %s", err)
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, japan)
	if err != nil {
		return fmt.Errorf("Failed to parse endDate: %s", err)
	}

	var logs chan storage.SCxLogLine = make(chan storage.SCxLogLine, 10)
	var errChan chan error = make(chan error)
	var finished chan int = make(chan int, 1)

	go archive.GrepLogs(lobby, users, start, end, logs, errChan, finished)

	if oFormat == "json" {
		fmt.Println("[")
	}

	var first bool = true
	for {
		select {
		case logLine := <-logs:
			if oFormat == "json" && !first {
				fmt.Println(",")
			}
			first = false
			err = outputLogLine(oFormat, logLine)
			if err != nil {
				return err
			}
		case err = <-errChan:
			return err
		case <-finished:
			if oFormat == "json" {
				fmt.Println("]")
			}
			return nil
		}
	}
}
