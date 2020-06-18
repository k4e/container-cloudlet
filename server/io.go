package main

import (
	"bufio"
	"errors"
	"io"
)

func Readline(reader io.Reader) ([]byte, error) {
	scan := bufio.NewScanner(reader)
	scanResult := scan.Scan()
	if err := scan.Err(); err != nil {
		return nil, err
	}
	if !scanResult {
		return nil, errors.New("Empty message received")
	}
	msg := scan.Bytes()
	return msg, nil
}
