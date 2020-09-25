package main

import (
	"errors"
	"net"
	"sync"
)

type ForwardingService struct {
	muxClose   sync.Mutex
	closed     bool
	network    string
	clientAddr *net.TCPAddr
	appAddr    *net.TCPAddr
	ln*        net.TCPListener
	sp         SessionPool
}

func StartForwardingService(network string, clientAddr, appAddr *net.TCPAddr) (*ForwardingService, error) {
	p := &ForwardingService{
		closed:     false,
		network:    network,
		clientAddr: clientAddr,
		appAddr:    appAddr,
		ln:         nil,
	}
	ln, err := net.ListenTCP(network, clientAddr)
	if err != nil {
		return nil, err
	}
	p.ln = ln
	go p.routine()
	return p, err
}

func (p *ForwardingService) Close() error {
	if p.ln == nil {
		return errors.New("Socket not listening")
	}
	p.muxClose.Lock()
	err := p.ln.Close()
	p.closed = true
	p.muxClose.Unlock()
	return err
}

func (p *ForwardingService) routine() {
	var dest string
	if p.appAddr != nil {
		dest = "app=" + p.appAddr.String()
	} else {
		dest = "?"
	}
	Logger.InfoF("Forwarding open: client=%s <--> %s\n", p.clientAddr.String(), dest)
	defer func() {
		Logger.InfoF("Forwarding closed: client=%s <--> %s\n", p.clientAddr.String(), dest)
	}()
	for {
		brk := false
		p.muxClose.Lock()
		brk = p.closed
		p.muxClose.Unlock()
		if brk {
			break
		}
		if err := p.sp.Accept(p.ln, p.network, p.appAddr); err != nil {
			Logger.ErrorE(err)
		}
	}
}
