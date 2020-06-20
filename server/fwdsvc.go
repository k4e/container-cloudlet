package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

const bufSize = 64 * 1024

type ForwardingService struct {
	muxStart   sync.Mutex
	muxClose   sync.Mutex
	network    string
	clientAddr string
	hostAddr   string
	ln         net.Listener
	started    bool
	closed     bool
}

func NewForwarding(network, clientAddr, hostAddr string) (*ForwardingService, error) {
	p := &ForwardingService{
		network:    network,
		clientAddr: clientAddr,
		hostAddr:   hostAddr,
		ln:         nil,
		started:    false,
		closed:     false,
	}
	ln, err := net.Listen(network, clientAddr)
	p.ln = ln
	return p, err
}

func (p *ForwardingService) Start() error {
	if p.ln == nil {
		return errors.New("Socket not listening")
	}
	var err error
	p.muxStart.Lock()
	if !p.started {
		p.started = true
		go p.routine()
	} else {
		err = errors.New("Service already started")
	}
	p.muxStart.Unlock()
	return err
}

func (p *ForwardingService) Close() error {
	if p.ln == nil {
		return errors.New("Socket not listening")
	}
	var err error
	p.muxClose.Lock()
	p.closed = true
	err = p.ln.Close()
	p.muxClose.Unlock()
	return err
}

func (p *ForwardingService) routine() {
	fmt.Printf("Forwarding started: %s -> %s\n", p.clientAddr, p.hostAddr)
	defer func() {
		fmt.Printf("Forwarding closed: %s -> %s\n", p.clientAddr, p.hostAddr)
	}()
	for {
		brk := false
		p.muxClose.Lock()
		brk = p.closed
		p.muxClose.Unlock()
		if brk {
			break
		}
		clientConn, err := p.ln.Accept()
		if err != nil {
			if !isClosedError(err) {
				p.printError(err)
			}
			continue
		}
		p.onAccept(clientConn)
	}
}

func (p *ForwardingService) onAccept(clientConn net.Conn) {
	var hostConn net.Conn
	close := func() {
		if clientConn != nil {
			if err := clientConn.Close(); err != nil {
				p.printError(err)
			}
		}
		if hostConn != nil {
			if err := hostConn.Close(); err != nil {
				p.printError(err)
			}
		}
	}
	hostConn, err := net.Dial(p.network, p.hostAddr)
	if err != nil {
		p.printError(err)
		return
	}
	wg := &sync.WaitGroup{}
	var once sync.Once
	stream := func(reader, writer net.Conn) {
		defer func() {
			once.Do(close)
			wg.Done()
		}()
		buf := make([]byte, bufSize)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				if err != io.EOF && !isClosedError(err) {
					p.printError(err)
				}
				return
			}
			if n > 0 {
				m, err := writer.Write(buf[:n])
				if err != nil {
					if err != io.EOF && !isClosedError(err) {
						p.printError(err)
					}
					return
				}
				if n != m {
					p.printError(errors.New(fmt.Sprintf("Read: %d, Write: %d", n, m)))
				}
			}
		}
	}
	wg.Add(1)
	go stream(clientConn, hostConn)
	wg.Add(1)
	go stream(hostConn, clientConn)
	wg.Wait()
}

func (p *ForwardingService) printError(e error) {
	PrintError(e)
}

func isClosedError(e error) bool {
	want := "use of closed network connection"
	return strings.Contains(e.Error(), want)
}
