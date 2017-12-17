package cmd

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/c-14/gtenlog/storage"
)

var grepUsage error = errors.New("usage: gtenlog grep [-s <date>] [-e <date>] [-a <userFile>] <lobby> <logRoot>")

func Grep(args []string) error {
	if len(args) < 2 {
		return grepUsage
	}
	var lobby string
	var startDate, endDate string
	var userPath string

	var grepFlags = flag.NewFlagSet("grep", flag.ExitOnError)
	grepFlags.StringVar(&startDate, "s", "2006-07-01", "First date for which to output data")
	grepFlags.StringVar(&endDate, "e", getDefaultEndDate(), "Last date for which to output data")
	grepFlags.StringVar(&userPath, "a", "", "Path to json file containing user/alias mapping")
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

	return archive.GrepLogs(lobby, users, start, end)
}
