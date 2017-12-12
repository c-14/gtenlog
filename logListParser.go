package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

type TokenType struct {
	File string
	Size int64
}

type LogListParser struct {
	r *bufio.Reader
	t TokenType
	e error
}

func (p *LogListParser) Err() error {
	return p.e
}

func InitLogListParser(r io.Reader) (p LogListParser, err error) {
	p.r = bufio.NewReader(r)
	err = p.consumeHeader()
	return
}

func (p *LogListParser) consumeHeader() error {
	header, err := p.r.ReadBytes('\n')
	if err != nil {
		return err
	}
	if bytes.Compare(header, []byte("list([\r\n")) != 0 {
		return fmt.Errorf("LogList begins with an invalid header: %s", header)
	}
	return nil
}

func (p *LogListParser) readLiteral(delim byte, check []byte) error {
	literal, err := p.r.ReadBytes(delim)
	if err != nil {
		return err
	}
	if bytes.Compare(literal, check) != 0 {
		return fmt.Errorf("Unexpected literal: %s, should have been: %s", literal, check)
	}
	return nil
}

func (p *LogListParser) readFileName() error {
	b, err := p.r.ReadByte()
	if err != nil {
		return err
	}
	if b != '\'' {
		return fmt.Errorf("Unexpected %v, expected '", b)
	}
	s, err := p.r.ReadBytes('\'')
	if err != nil {
		return err
	}
	p.t.File = string(s[:len(s)-1])
	return nil
}

func (p *LogListParser) readSize() error {
	s, err := p.r.ReadBytes('}')
	if err != nil {
		return err
	}
	size, err := strconv.ParseInt(string(s[:len(s)-1]), 10, 32)
	if err != nil {
		return fmt.Errorf("Failed parsing size: %s", err)
	}
	p.t.Size = size
	return nil
}

func (p *LogListParser) consumeLog() (ret bool, err error) {
	err = p.readLiteral(':', []byte("{file:"))
	if err != nil {
		return
	}
	err = p.readFileName()
	if err != nil {
		return
	}
	err = p.readLiteral(':', []byte(",size:"))
	if err != nil {
		return
	}
	err = p.readSize()
	if err != nil {
		return
	}

	b, err := p.r.ReadBytes('\n')
	if err != nil {
		return
	}
	if bytes.Compare(b, []byte(",\r\n")) == 0 {
		return true, nil
	}
	return false, nil
}

func (p *LogListParser) Scan() bool {
	prog, err := p.consumeLog()
	if err != nil {
		p.e = err
		return false
	}
	return prog
}

func (p *LogListParser) Token() TokenType {
	return p.t
}
