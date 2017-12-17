package storage

import (
	"bufio"
	"path/filepath"
	"os"
)

type LogSet map[string]bool

type UserLogSet struct {
	Path string
	User string
	Logs LogSet
}

func (l UserLogSet) Write() error {
	fPath := filepath.Join(l.Path, "user", l.User, "localLogs.index")
	file, err := os.OpenFile(fPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(fPath), 0755)
		if err != nil {
			return err
		}
		return l.Write()
	} else if err != nil {
		return err
	}
	defer file.Close()

	wrLines := bufio.NewWriter(file)
	for data, _ := range l.Logs {
		_, err = wrLines.WriteString(data)
		if err != nil {
			return err
		}
		err = wrLines.WriteByte(0x0A)
		if err != nil {
			return err
		}
	}
	return wrLines.Flush()
}

func (l *UserLogSet) Read() error {
	fPath := filepath.Join(l.Path, "user", l.User, "localLogs.index")
	file, err := os.Open(fPath)
	if err != nil {
		return err
	}
	defer file.Close()

	lines := bufio.NewScanner(file)
	for lines.Scan() {
		s := lines.Text()
		l.Logs[s] = true
	}
	if err = lines.Err(); err != nil {
		return err
	}

	return nil
}
