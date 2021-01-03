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
	Fwdr_BufferSize = 32 * 1024 * 1024
)

type Forwarder struct {
	clientConn         *net.TCPConn
	serverConn         *net.TCPConn
	muxs               [2]sync.Mutex
	isUpClosed         bool
	isDownClosed       bool
	chanClosed         chan struct{}
	dataRate           int
	chanEmergencyClose chan struct{}
}

func NewForwarder() *Forwarder {
	fwdr := &Forwarder{}
	fwdr.chanClosed = make(chan struct{})
	return fwdr
}

func (p *Forwarder) SetDataRate(dataRate int) {
	p.dataRate = dataRate
	if p.dataRate > 0 {
		if p.clientConn != nil {
			// bufsz := MinInt(p.dataRate*500, math.MaxInt32)
			// p.clientConn.SetReadBuffer(bufsz)
			p.clientConn.SetLinger(0)
		}
	}
}

func (p *Forwarder) Accept(network string, serverAddr *net.TCPAddr, clientConn *net.TCPConn) {
	defer close(p.chanClosed)
	p.clientConn = clientConn
	if p.dataRate > 0 {
		// bufsz := MinInt(p.dataRate*500, math.MaxInt32)
		// p.clientConn.SetReadBuffer(bufsz)
		p.clientConn.SetLinger(0)
		p.chanEmergencyClose = make(chan struct{})
	}
	if serverConn, err := p.dialTCP(network, serverAddr); err != nil {
		Logger.ErrorE(err)
		if err := p.clientConn.Close(); err != nil {
			Logger.Warn("[Fwd] clientConn.Close: " + err.Error())
		}
		return
	} else {
		p.serverConn = serverConn
	}
	Logger.DebugF("[Fwd] Open: %s <--> %s\n",
		p.clientConn.RemoteAddr().String(), p.serverConn.RemoteAddr().String())
	defer func() {
		Logger.DebugF("[Fwd] Close: %s <--> %s\n",
			p.clientConn.RemoteAddr().String(), p.serverConn.RemoteAddr().String())
	}()
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.upstream(p.clientConn, p.serverConn, wg)
	go p.downstream(p.serverConn, p.clientConn, wg)
	wg.Wait()
	if err := p.clientConn.Close(); err != nil {
		Logger.Warn("[Fwd] clientConn.Close: " + err.Error())
	}
	if err := p.serverConn.Close(); err != nil {
		Logger.Warn("[Fwd] serverConn.Close: " + err.Error())
	}
}

func (p *Forwarder) Close() {
	p.closeUpstream()
	<-p.chanClosed
}

func (p *Forwarder) dialTCP(network string, serverAddr *net.TCPAddr) (*net.TCPConn, error) {
	conn, err := net.DialTCP(network, nil, serverAddr)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return conn, nil
}

func (p *Forwarder) closeUpstream() {
	p.muxs[0].Lock()
	defer p.muxs[0].Unlock()
	if p.isUpClosed {
		return
	}
	if p.chanEmergencyClose != nil {
		close(p.chanEmergencyClose)
	}
	if err := p.clientConn.CloseRead(); err != nil {
		Logger.Warn("[Fwd] clientConn.CloseRead: " + err.Error())
	}
	p.isUpClosed = true
}

func (p *Forwarder) closeDownstream() {
	p.muxs[1].Lock()
	defer p.muxs[1].Unlock()
	if p.isDownClosed {
		return
	}
	if err := p.serverConn.CloseRead(); err != nil {
		Logger.Warn("[Fwd] serverConn.CloseRead: " + err.Error())
	}
	p.isDownClosed = true
}

func (p *Forwarder) upstream(clientIn io.Reader, serverOut io.Writer, wg *sync.WaitGroup) {
	defer func() {
		Logger.Debug("[Fwd] Upstream: finish")
		p.closeDownstream()
		if wg != nil {
			wg.Done()
		}
	}()
	if p.dataRate > 0 {
		Logger.InfoF("[Fwd] Upstream: data rate limited to %d Mbps\n", p.dataRate)
	}
	buf := make([]byte, Fwdr_BufferSize)
	for {
		select {
		case <-p.chanEmergencyClose:
			return
		default:
		}
		nr, cerr := clientIn.Read(buf)
		if nr > 0 {
			timeWriteStart := time.Now()
			nw, serr := serverOut.Write(buf[0:nr])
			if p.dataRate > 0 {
				timeWrite := time.Now().Sub(timeWriteStart)
				dataBytes := float64(nw)
				rateBps := float64(p.dataRate) * 125000.0
				timeToSleep := (dataBytes / rateBps) - timeWrite.Seconds()
				if timeToSleep > 0 {
					time.Sleep(time.Duration(timeToSleep*1000000000) * time.Nanosecond)
				}
			}
			if serr != nil {
				if IsClosedError(serr) {
					Logger.Warn("[Fwd] Upstream: server is close")
				} else {
					Logger.ErrorE(errors.WithStack(serr))
				}
				if nr != nw {
					Logger.ErrorE(errors.WithStack(io.ErrShortWrite))
				}
				return
			} else {
				Logger.TraceF("[Fwd] Upstream: wrote: %s", buf[0:nw])
			}
		}
		if cerr != nil {
			if (cerr == io.EOF) || IsClosedError(cerr) {
				Logger.Debug("[Fwd] Upstream: client reached end")
			} else {
				Logger.ErrorE(errors.WithStack(cerr))
			}
			return
		}
	}
}

func (p *Forwarder) downstream(serverIn io.Reader, clientOut io.Writer, wg *sync.WaitGroup) {
	defer func() {
		Logger.Debug("[Fwd] Downstream: finish")
		p.closeUpstream()
		if wg != nil {
			wg.Done()
		}
	}()
	buf := make([]byte, Fwdr_BufferSize)
	for {
		nr, serr := serverIn.Read(buf)
		if nr > 0 {
			nw, cerr := clientOut.Write(buf[0:nr])
			if cerr != nil {
				if IsClosedError(cerr) {
					Logger.Warn("[Fwd] Downstream: client is close")
				} else {
					Logger.ErrorE(errors.WithStack(cerr))
				}
				if nr != nw {
					Logger.ErrorE(errors.WithStack(io.ErrShortWrite))
				}
				return
			} else {
				Logger.TraceF("[Fwd] Downstream: wrote: %s\n", buf[0:nw])
			}
		}
		if serr != nil {
			if (serr == io.EOF) || IsClosedError(serr) {
				Logger.Debug("[Fwd] Downstream: server is close")
			} else {
				Logger.ErrorE(serr)
			}
			return
		}
	}
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
