package main

import (
	"net"

	"github.com/pkg/errors"
)

type ForwarderService struct {
	network    string
	clientAddr *net.TCPAddr
	serverAddr *net.TCPAddr
	isExtHost  bool
	closeChan  chan struct{}
}

func StartForwarderService(
	network string,
	clientAddr *net.TCPAddr,
	serverAddr *net.TCPAddr,
	isExtHost bool,
) (*ForwarderService, error) {
	closeChan := make(chan struct{})
	p := &ForwarderService{
		network:    network,
		clientAddr: clientAddr,
		serverAddr: serverAddr,
		isExtHost:  isExtHost,
		closeChan:  closeChan,
	}
	ln, err := net.ListenTCP(network, clientAddr)
	if err != nil {
		return nil, err
	}
	go func() {
		<-p.closeChan
		if err := ln.Close(); err != nil {
			Logger.ErrorE(errors.WithStack(err))
		}
	}()
	go p.listener(ln)
	return p, err
}

func (p *ForwarderService) Close() error {
	close(p.closeChan)
	return nil
}

func (p *ForwarderService) listener(ln *net.TCPListener) {
	Logger.InfoF("Forwarding open: %s <--> %s\n", p.clientAddr.String(), p.serverAddr.String())
	defer func() {
		Logger.InfoF("Forwarding closed: %s <--> %s\n", p.clientAddr.String(), p.serverAddr.String())
	}()
	for {
		select {
		case <-p.closeChan:
			return
		default:
		}
		if err := AcceptForwarder(ln, p.network, p.serverAddr, p.isExtHost); err != nil {
			Logger.ErrorE(err)
		}
	}
}
