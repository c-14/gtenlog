package cmd

import (
	"time"
)

func getDefaultStartDate() string {
	japan, _ := time.LoadLocation("Japan")
	return time.Now().In(japan).AddDate(0, 0, -8).Format("2006-01-02")
}

func getDefaultEndDate() string {
	japan, _ := time.LoadLocation("Japan")
	return time.Now().In(japan).Format("2006-01-02")
}
