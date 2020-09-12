package main

import (
	"fmt"
	"os"
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
	fmt.Fprintln(os.Stderr, s)
}

func (p *SLog) ErrorF(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func (p *SLog) ErrorE(e error) {
	fmt.Fprintln(os.Stderr, e)
}

func (p *SLog) Warn(s string) {
	fmt.Println(s)
}

func (p *SLog) WarnF(format string, a ...interface{}) {
	fmt.Printf(format, a...)
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
