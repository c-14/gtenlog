package storage

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type SCxLog struct {
	Path string
	Date time.Time

	file *os.File
	gzLog *gzip.Reader
	lines *bufio.Scanner

	token SCxLogLine
	err error
}

type SCxLogLine interface {
	Parse(data string, date time.Time) error
}

type SCALogLine struct {
	Lobby string
	StartTime time.Time
	GameMode string
	Score []UserScore
}

type UserScore struct {
	UserName string
	Score float32
}

func parseUserScores(data string, numUsers int) ([]UserScore, error) {
	fields := strings.Split(data, " ")
	if len(fields) != numUsers {
		return []UserScore{}, errors.New("Invalid number of users")
	}

	scores := make([]UserScore, numUsers) 
	for i, field := range(fields) {
		scoreIndex := strings.LastIndexByte(field, '(')
		scores[i].UserName = field[0:scoreIndex]

		var err error
		var ts float64
		commaIndex := strings.LastIndexByte(field[scoreIndex + 1:], ',')
		if commaIndex == -1 {
			ts, err = strconv.ParseFloat(field[scoreIndex + 1:len(field) - 1], 32)
		} else {
			ts, err = strconv.ParseFloat(field[scoreIndex + 1:scoreIndex + commaIndex], 32)
		}
		if err != nil {
			return scores, err
		}
		scores[i].Score = float32(ts)
	}

	return scores, nil
}

func (ll *SCALogLine) Parse(data string, date time.Time) error {
	fields := strings.Split(data, " | ")
	if len(fields) != 4 {
		return fmt.Errorf("Error while parsing line; expected 4 fields, got %v", len(fields))
	}

	var err error
	ll.Lobby = fields[0]
	ll.GameMode = fields[2]

	start, err := time.Parse("15:04", fields[1])
	if err != nil {
		return err
	}
	ll.StartTime = date.Add(time.Hour * time.Duration(start.Hour()) + time.Minute * time.Duration(start.Minute()))
	ll.Score, err = parseUserScores(fields[3], getNumPlayers(ll.GameMode))

	return err
}

func (ll *SCALogLine) String() string {
	var b strings.Builder
	b.WriteString(ll.Lobby)
	b.WriteString(" | ")
	b.WriteString(ll.StartTime.Format("15:04"))
	b.WriteString(" | ")
	b.WriteString(ll.GameMode)
	b.WriteString(" |")
	for _, s := range(ll.Score) {
		b.WriteByte(' ')
		b.WriteString(s.UserName)
		b.WriteByte('(')
		b.WriteString(strconv.FormatFloat(float64(s.Score), 'f', 1, 32))
		b.WriteByte(')')
	}
	return b.String()
}

func InitSCxLogParser(path string) (SCxLog, error) {
	var s SCxLog

	s.Path = path
	basePath := filepath.Base(path)

	japan, _ := time.LoadLocation("Japan")

	var err error
	switch scx := basePath[:3]; scx {
	case "sca":
		s.token = &SCALogLine{}
		s.Date, err = time.ParseInLocation("20060102", filepath.Base(path)[3:11], japan)
	default:
		return s, fmt.Errorf("Log Type %s not yet implemented", scx)
	}

	if err != nil {
		return s, err
	}

	return s, s.Open()
}

func (s *SCxLog) Open() error {
	var err error

	s.file, err = os.Open(s.Path)
	if err != nil {
		return err
	}

	s.gzLog, err = gzip.NewReader(s.file)
	if err != nil {
		s.file.Close()
		return err
	}

	s.lines = bufio.NewScanner(s.gzLog)

	return nil
}

func (s SCxLog) Close() error {
	err := s.gzLog.Close()
	if err != nil {
		s.file.Close()
		return err
	}

	return s.file.Close()
}

func (s *SCxLog) Scan() bool {
	if (!s.lines.Scan()) {
		return false
	}
	
	err := s.token.Parse(s.lines.Text(), s.Date)
	if err != nil {
		s.err = err
		return false
	}
	return true
}

func (s SCxLog) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.lines.Err()
}

func (s SCxLog) Token() SCxLogLine {
	return s.token
}

func getNumPlayers(gameMode string) int {
	if r, _ := utf8.DecodeRuneInString(gameMode); r == '三' {
		return 3
	} else if r == '四' {
		return 4
	} else {
		return -1
	}
}
