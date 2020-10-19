package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type SLog struct {
	verbose bool
}

var Logger = SLog{
	verbose: false,
}

func (p *SLog) SetVerbosity(verbose bool) {
	p.verbose = verbose
}

func (p *SLog) Error(s string) {
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d; ", filepath.Base(file), line)
	}
	fmt.Fprintln(os.Stderr, pos+s)
}

func (p *SLog) ErrorF(format string, a ...interface{}) {
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d; ", filepath.Base(file), line)
	}
	fmt.Fprintf(os.Stderr, pos+format, a...)
}

func (p *SLog) ErrorE(e error) {
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d; ", filepath.Base(file), line)
	}
	fmt.Fprintln(os.Stderr, pos+e.Error())
}

func (p *SLog) Warn(s string) {
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d; ", filepath.Base(file), line)
	}
	fmt.Println(pos + s)
}

func (p *SLog) WarnF(format string, a ...interface{}) {
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d; ", filepath.Base(file), line)
	}
	fmt.Printf(pos+format, a...)
}

func (p *SLog) Info(s string) {
	fmt.Println(s)
}

func (p *SLog) InfoF(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

func (p *SLog) Debug(s string) {
	if p.verbose {
		fmt.Println(s)
	}
}

func (p *SLog) DebugF(format string, a ...interface{}) {
	if p.verbose {
		fmt.Printf(format, a...)
	}
}
