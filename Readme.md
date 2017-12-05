# gtenlog

Go Tenhou Log Fetcher/Analyzer

A simple program to fetch logs from tenhou and/or analyze them.

Current Status:

	Release - 0.1.0-alpha
	Bugs - Probably

## Download

	go get github.com/c-14/gtenlog

## Usage

* Scrape logs from Browser/App database
```
gtenlog scrape <webappstore.sqlite> <output_path>
```

* Fetch details from tenhou.net for scraped logs
```
gtenlog fetch user <log_root>
```

* Fetch archived logs for the past 9 days
```
gtenlog fetch daily <log_root>
```
