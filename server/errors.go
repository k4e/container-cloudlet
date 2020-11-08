package main

import (
	stderrors "errors"
	"net"
	"os"
	"strings"
)

func IsClosedError(e error) bool {
	msg := "use of closed network connection"
	return strings.Contains(e.Error(), msg)
}

func IsDeadlineExceeded(err error) bool {
	nerr, ok := err.(net.Error)
	if !ok {
		return false
	}
	if !nerr.Timeout() {
		return false
	}
	if !stderrors.Is(err, os.ErrDeadlineExceeded) {
		return false
	}
	return true
}
