package main

import (
	"container/list"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	FwdSvc_LnTimeoutDuration = 10 * time.Millisecond
)

type ForwarderService struct {
	network     string
	clientAddr  *net.TCPAddr
	serverAddr  *net.TCPAddr
	isExtHost   bool
	muxFwdrs    sync.Mutex
	fwdrs       *list.List
	condSuspend *sync.Cond
	chanSuspend chan struct{}
	isSuspended bool
	chanClose   chan struct{}
}

func StartForwarderService(
	network string,
	clientAddr *net.TCPAddr,
	serverAddr *net.TCPAddr,
	isExtHost bool,
) (*ForwarderService, error) {
	fwdrs := list.New()
	condSuspend := sync.NewCond(&sync.Mutex{})
	chanSuspend := make(chan struct{}, 1)
	chanClose := make(chan struct{})
	p := &ForwarderService{
		network:     network,
		clientAddr:  clientAddr,
		serverAddr:  serverAddr,
		isExtHost:   isExtHost,
		fwdrs:       fwdrs,
		condSuspend: condSuspend,
		chanSuspend: chanSuspend,
		chanClose:   chanClose,
	}
	ln, err := net.ListenTCP(network, clientAddr)
	if err != nil {
		return nil, err
	}
	go func() {
		<-p.chanClose
		if err := ln.Close(); err != nil {
			Logger.Warn("[Fwdsvc] ln.Close: " + err.Error())
		}
	}()
	go p.listener(ln)
	return p, err
}

func (p *ForwarderService) Close() error {
	close(p.chanClose)
	return nil
}

func (p *ForwarderService) Suspend() {
	p.condSuspend.L.Lock()
	p.isSuspended = true
	p.condSuspend.Broadcast()
	p.condSuspend.L.Unlock()
	<-p.chanSuspend
}

func (p *ForwarderService) Resume() {
	p.condSuspend.L.Lock()
	p.isSuspended = false
	p.condSuspend.Broadcast()
	p.condSuspend.L.Unlock()
}

func (p *ForwarderService) CloseAllForwarders() {
	if !p.isSuspended {
		panic("Must be suspended before CloseAllForwarders")
	}
	wg := sync.WaitGroup{}
	p.muxFwdrs.Lock()
	for e := p.fwdrs.Front(); e != nil; e = e.Next() {
		fwdr := e.Value.(*Forwarder)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fwdr.Close()
		}()
	}
	p.muxFwdrs.Unlock()
	wg.Wait()
}

func (p *ForwarderService) ChangeServerAddr(serverAddr *net.TCPAddr) {
	if !p.isSuspended {
		panic("Must be suspended before ChangeServerAddr")
	}
	p.serverAddr = serverAddr
}

func (p *ForwarderService) listener(ln *net.TCPListener) {
	for {
		p.condSuspend.L.Lock()
		if p.isSuspended {
			Logger.Debug("[Fwdsvc] Suspended")
			p.chanSuspend <- struct{}{}
			for p.isSuspended {
				p.condSuspend.Wait()
			}
		}
		p.condSuspend.L.Unlock()
		if err := ln.SetDeadline(time.Now().Add(FwdSvc_LnTimeoutDuration)); err != nil {
			Logger.Warn("[Fwdsvc] ln.SetDeadline: " + err.Error())
		}
		clientConn, err := ln.AcceptTCP()
		if err != nil {
			if IsDeadlineExceeded(err) {
				continue
			} else if IsClosedError(err) {
				Logger.Debug("[Fwdsvc] Accept returned with close")
				return
			} else {
				Logger.ErrorE(errors.WithStack(err))
				return
			}
		}
		Logger.Debug("[Fwdsvc] Accept client conn: " + clientConn.RemoteAddr().String())
		fwdr := NewForwarder(p.network, p.serverAddr, clientConn)
		p.muxFwdrs.Lock()
		elem := p.fwdrs.PushBack(fwdr)
		p.muxFwdrs.Unlock()
		go func() {
			fwdr.Open()
			p.muxFwdrs.Lock()
			p.fwdrs.Remove(elem)
			p.muxFwdrs.Unlock()
			Logger.Debug("[Fwdsvc] Close client conn: " + clientConn.RemoteAddr().String())
			if err := clientConn.Close(); err != nil {
				Logger.Warn("[Fwdsvc] clientConn.Close: " + err.Error())
			}
		}()
	}
}
