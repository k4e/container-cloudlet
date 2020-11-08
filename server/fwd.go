package main

/*
  TODO:
  - Forwarder.Start() のロック内での Dial() や Close() は時間がかかりすぎるので回避する
  - serverConn の保持によるコネクションの維持を実現する
*/

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	Fwdr_ConnReadTimeoutDuration = 10 * time.Millisecond
	Fwdr_BufferSize              = 32 * 1024 * 1024
)

type Forwarder struct {
	initParam struct {
		network    string
		serverAddr *net.TCPAddr
		clientConn *net.TCPConn
	}
	muxStream               sync.Mutex
	upAlive                 bool
	downAlive               bool
	chanClosed              chan struct{}
	condUsingStreamConn     *sync.Cond
	cntUsingStreamConn      int
	condSuspendedStreamConn *sync.Cond
	isSuspendedStreamConn   bool
}

func NewForwarder(
	network string,
	serverAddr *net.TCPAddr,
	clientConn *net.TCPConn,
) *Forwarder {
	fwdr := &Forwarder{}
	fwdr.initParam.network = network
	fwdr.initParam.serverAddr = serverAddr
	fwdr.initParam.clientConn = clientConn
	fwdr.chanClosed = make(chan struct{})
	fwdr.condUsingStreamConn = sync.NewCond(&sync.Mutex{})
	fwdr.condSuspendedStreamConn = sync.NewCond(&sync.Mutex{})
	return fwdr
}

func (p *Forwarder) Open() {
	clientConn := p.initParam.clientConn
	defer func() {
		Logger.DebugF("[Fwd] Close: %s <--> %s\n",
			clientConn.RemoteAddr().String(), p.initParam.serverAddr.String())
	}()
	Logger.DebugF("[Fwd] Open: %s <--> %s\n",
		clientConn.RemoteAddr().String(), p.initParam.serverAddr.String())
	p.setStreamsAlive(false)
	serverConn, err := p.dialTCP(p.initParam.network, p.initParam.serverAddr)
	if err != nil {
		Logger.ErrorE(err)
		Logger.Debug("[Fwd] Client conn close")
		if err := clientConn.Close(); err != nil {
			Logger.Warn("[Fwd] clientConn.Close: " + err.Error())
		}
		return
	}
	Logger.Debug("[Fwd] Server conn open")
	p.setStreamsAlive(true)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.upstream(clientConn, serverConn, wg)
	go p.downstream(clientConn, serverConn, wg)
	wg.Wait()
	if err := serverConn.Close(); err != nil {
		Logger.Warn("[Fwd] serverConn.Close: " + err.Error())
	}
	close(p.chanClosed)
}

func (p *Forwarder) Close() {
	p.SuspendStream()
	p.setStreamsAlive(false)
	p.ResumeStream()
	<-p.chanClosed
}

func (p *Forwarder) SuspendStream() {
	p.condSuspendedStreamConn.L.Lock()
	p.isSuspendedStreamConn = true
	p.condSuspendedStreamConn.Broadcast()
	p.condSuspendedStreamConn.L.Unlock()
	p.condUsingStreamConn.L.Lock()
	for p.cntUsingStreamConn > 0 {
		p.condUsingStreamConn.Wait()
	}
	p.condUsingStreamConn.L.Unlock()
}

func (p *Forwarder) ResumeStream() {
	p.condSuspendedStreamConn.L.Lock()
	p.isSuspendedStreamConn = false
	p.condSuspendedStreamConn.Broadcast()
	p.condSuspendedStreamConn.L.Unlock()
}

func (p *Forwarder) dialTCP(network string, serverAddr *net.TCPAddr) (*net.TCPConn, error) {
	conn, err := net.DialTCP(network, nil, serverAddr)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return conn, nil
}

func (p *Forwarder) enterStreamConnSection() bool {
	p.condSuspendedStreamConn.L.Lock()
	for p.isSuspendedStreamConn {
		p.condSuspendedStreamConn.Wait()
	}
	p.condSuspendedStreamConn.L.Unlock()
	p.condUsingStreamConn.L.Lock()
	p.cntUsingStreamConn++
	if p.cntUsingStreamConn > 2 {
		panic("number of routines using stream conn > 2")
	}
	p.condUsingStreamConn.Broadcast()
	p.condUsingStreamConn.L.Unlock()
	p.muxStream.Lock()
	if !p.upAlive || !p.downAlive {
		p.muxStream.Unlock()
		return false
	}
	p.muxStream.Unlock()
	return true
}

func (p *Forwarder) leaveStreamConnSection() {
	p.condUsingStreamConn.L.Lock()
	p.cntUsingStreamConn--
	if p.cntUsingStreamConn < 0 {
		panic("number of routines using stream conn < 0")
	}
	p.condUsingStreamConn.Broadcast()
	p.condUsingStreamConn.L.Unlock()
}

func (p *Forwarder) setStreamsAlive(b bool) {
	defer p.muxStream.Unlock()
	p.muxStream.Lock()
	p.upAlive = b
	p.downAlive = b
}

func (p *Forwarder) setUpstreamAlive(b bool) {
	defer p.muxStream.Unlock()
	p.muxStream.Lock()
	p.upAlive = b
}

func (p *Forwarder) setDownstreamAlive(b bool) {
	defer p.muxStream.Unlock()
	p.muxStream.Lock()
	p.downAlive = b
}

func (p *Forwarder) upstream(clientConn, serverConn *net.TCPConn, wg *sync.WaitGroup) {
	defer func() {
		Logger.Debug("[Fwd] Upstream: finish")
		p.setUpstreamAlive(false)
		if wg != nil {
			wg.Done()
		}
	}()
	buf := make([]byte, Fwdr_BufferSize)
	for {
		if !p.enterStreamConnSection() {
			p.leaveStreamConnSection()
			return
		}
		if err := clientConn.SetReadDeadline(time.Now().Add(Fwdr_ConnReadTimeoutDuration)); err != nil {
			Logger.Warn("[Fwd] Upstream: clientConn.SetReadDeadline: " + err.Error())
		}
		nr, cerr := clientConn.Read(buf)
		flagReturn := false
		if nr > 0 {
			nw, serr := serverConn.Write(buf[0:nr])
			if serr != nil {
				if IsClosedError(serr) {
					Logger.Debug("[Fwd] Upstream: server is close")
				} else if nr != nw {
					Logger.ErrorE(errors.WithStack(io.ErrShortWrite))
				} else {
					Logger.ErrorE(errors.WithStack(serr))
				}
				flagReturn = true
			} else {
				Logger.TraceF("[Fwd] Upstream: wrote: %s", buf[0:nw])
			}
		}
		p.leaveStreamConnSection()
		if cerr != nil {
			if IsDeadlineExceeded(cerr) {
				continue
			}
			if (cerr == io.EOF) || IsClosedError(cerr) {
				Logger.Debug("[Fwd] Upstream: client reached end")
			} else {
				Logger.ErrorE(errors.WithStack(cerr))
			}
			return
		}
		if flagReturn {
			return
		}
	}
}

func (p *Forwarder) downstream(clientConn, serverConn *net.TCPConn, wg *sync.WaitGroup) {
	defer func() {
		Logger.Debug("[Fwd] Downstream: finish")
		p.setDownstreamAlive(false)
		if wg != nil {
			wg.Done()
		}
	}()
	buf := make([]byte, Fwdr_BufferSize)
	for {
		if !p.enterStreamConnSection() {
			p.leaveStreamConnSection()
			return
		}
		if err := serverConn.SetReadDeadline(time.Now().Add(Fwdr_ConnReadTimeoutDuration)); err != nil {
			Logger.Warn("[Fwd] Downstream: serverConn.SetReadDeadline: " + err.Error())
		}
		nr, serr := serverConn.Read(buf)
		flagReturn := false
		if nr > 0 {
			nw, cerr := clientConn.Write(buf[0:nr])
			if cerr != nil {
				if IsClosedError(cerr) {
					Logger.Debug("[Fwd] Downstream: client is close")
				} else if nr != nw {
					Logger.ErrorE(errors.WithStack(io.ErrShortWrite))
				} else {
					Logger.ErrorE(errors.WithStack(cerr))
				}
				flagReturn = true
			} else {
				Logger.TraceF("[Fwd] Downstream: wrote: %s\n", buf[0:nw])
			}
		}
		p.leaveStreamConnSection()
		if serr != nil {
			if IsDeadlineExceeded(serr) {
				continue
			}
			if (serr == io.EOF) || IsClosedError(serr) {
				Logger.Debug("[Fwd] Downstream: server reached end")
			} else {
				Logger.ErrorE(serr)
			}
			return
		}
		if flagReturn {
			return
		}
	}
}
