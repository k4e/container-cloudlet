package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	VerbosityInfo  = 0
	VerbosityDebug = 1
	VerbosityTrace = 2
)

type SLog struct {
	Verbosity int
}

var Logger = SLog{
	Verbosity: VerbosityInfo,
}

func (p *SLog) SetVerbosity(verbosity int) {
	p.Verbosity = verbosity
}

func (p *SLog) Error(s string) {
	col := "\x1b[31m[ERROR]\x1b[0m "
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d\x1b[31m::\x1b[0m ", filepath.Base(file), line)
	}
	fmt.Fprintln(os.Stderr, col+pos+s)
}

func (p *SLog) ErrorF(format string, a ...interface{}) {
	col := "\x1b[31m[ERROR]\x1b[0m "
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d\x1b[31m::\x1b[0m ", filepath.Base(file), line)
	}
	fmt.Fprintf(os.Stderr, col+pos+format, a...)
}

func (p *SLog) ErrorE(e error) {
	col := "\x1b[31m[ERROR]\x1b[0m "
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d\x1b[31m::\x1b[0m ", filepath.Base(file), line)
	}
	fmt.Fprintf(os.Stderr, col+pos+"%+v\n", e)
}

func (p *SLog) Warn(s string) {
	col := "\x1b[33m[WARN]\x1b[0m "
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d\x1b[33m::\x1b[0m ", filepath.Base(file), line)
	}
	fmt.Println(col + pos + s)
}

func (p *SLog) WarnF(format string, a ...interface{}) {
	col := "\x1b[33m[WARN]\x1b[0m "
	pos := ""
	if _, file, line, ok := runtime.Caller(1); ok {
		pos = fmt.Sprintf("%s:%d\x1b[33m::\x1b[0m ", filepath.Base(file), line)
	}
	fmt.Printf(col+pos+format, a...)
}

func (p *SLog) Info(s string) {
	col := "\x1b[36m[INFO]\x1b[0m "
	fmt.Println(col + s)
}

func (p *SLog) InfoF(format string, a ...interface{}) {
	col := "\x1b[36m[INFO]\x1b[0m "
	fmt.Printf(col+format, a...)
}

func (p *SLog) Debug(s string) {
	col := "\x1b[34m[DEBUG]\x1b[0m "
	if p.Verbosity >= VerbosityDebug {
		fmt.Println(col + s)
	}
}

func (p *SLog) DebugF(format string, a ...interface{}) {
	col := "\x1b[34m[DEBUG]\x1b[0m "
	if p.Verbosity >= VerbosityDebug {
		fmt.Printf(col+format, a...)
	}
}

func (p *SLog) Trace(s string) {
	col := "\x1b[34m[TRACE]\x1b[0m "
	if p.Verbosity >= VerbosityTrace {
		fmt.Println(col + s)
	}
}

func (p *SLog) TraceF(format string, a ...interface{}) {
	col := "\x1b[34m[TRACE]\x1b[0m "
	if p.Verbosity >= VerbosityTrace {
		fmt.Printf(col+format, a...)
	}
}
