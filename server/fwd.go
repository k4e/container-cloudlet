package main

/*
  TODO:
  - Forwarder.Start() のロック内での Dial() や Close() は時間がかかりすぎるので回避する
  - serverConn の保持によるコネクションの維持を実現する
*/

import (
	stderrors "errors"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const KeepSec = 60.
const TimeoutDuration = 1000 * time.Millisecond
const RetryInterval = 1000 * time.Millisecond
const BufferSize = 32 * 1024 * 1024

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

type Forwarder struct {
	keep       bool
	serverConn *ConnClosable
	muxStream  sync.Mutex
	upAlive    bool
	upBuf      []byte
	upBufLB    int
	upBufUB    int
	downAlive  bool
	downBuf    []byte
	downBufLB  int
	downBufUB  int
}

func AcceptForwarder(
	ln *net.TCPListener,
	network string,
	serverAddr *net.TCPAddr,
	isExtHost bool,
) error {
	if serverAddr == nil {
		return errors.New("serverAddr is empty")
	}
	clientConn, err := ln.AcceptTCP()
	if err != nil {
		return err
	}
	Logger.Debug("Accept: " + clientConn.RemoteAddr().String())
	fwd := newForwarder()
	go fwd.start(NewConnClosable(clientConn), network, serverAddr, false)
	return nil
}

func newForwarder() *Forwarder {
	p := &Forwarder{}
	p.upBuf = make([]byte, BufferSize)
	p.downBuf = make([]byte, BufferSize)
	return p
}

func (p *Forwarder) start(
	clientConn *ConnClosable,
	network string,
	serverAddr *net.TCPAddr,
	resume bool,
) {
	p.setStreamsAlive(false)
	var serverConn *ConnClosable
	if true {
		conn, err := net.DialTCP(network, nil, serverAddr)
		if err != nil {
			Logger.ErrorE(err)
			Logger.Debug("Client conn close")
			if err := clientConn.Close(); err != nil {
				Logger.Warn("Warning: clientConn.Close: " + err.Error())
			}
			return
		}
		Logger.Debug("Server conn open")
		serverConn = NewConnClosable(conn)
	}
	p.setStreamsAlive(true)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.upstream(clientConn, serverConn, wg)
	go p.downstream(clientConn, serverConn, wg)
	wg.Wait()
	clientConn.Close()
	serverConn.Close()
}

func (p *Forwarder) getStreamsAlive() bool {
	defer p.muxStream.Unlock()
	p.muxStream.Lock()
	return p.upAlive && p.downAlive
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

func (p *Forwarder) upstream(clientConn, serverConn *ConnClosable, wg *sync.WaitGroup) {
	defer func() {
		Logger.Debug("upstream: finish")
		p.setUpstreamAlive(false)
		if wg != nil {
			wg.Done()
		}
	}()
	// ct := &CountdownTimer{Deadline: KeepSec}
	for p.getStreamsAlive() {
		var cerr error
		if p.upBufUB == 0 {
			if err := clientConn.SetReadDeadline(time.Now().Add(TimeoutDuration)); err != nil {
				Logger.Warn("Warning: clientConn.SetReadDeadline: " + err.Error())
			}
			p.upBufUB, cerr = clientConn.Read(p.upBuf)
			p.upBufLB = 0
		}
		if p.upBufUB > 0 {
			for p.getStreamsAlive() {
				if serverConn.IsClosed() {
					return
				}
				if err := serverConn.SetWriteDeadline(time.Now().Add(TimeoutDuration)); err != nil {
					Logger.Warn("Warning: serverConn.SetWriteDeadline: " + err.Error())
				}
				m, herr := serverConn.Write(p.upBuf[p.upBufLB:p.upBufUB])
				Logger.TraceF("upstream: Wrote: %s", p.upBuf[p.upBufLB:p.upBufUB])
				p.upBufLB += m
				if herr != nil {
					if IsDeadlineExceeded(herr) {
						continue
					} else {
						return
					}
				}
				break
			}
			p.upBufUB = 0
			p.upBufLB = 0
		}
		if cerr != nil {
			if IsDeadlineExceeded(cerr) {
				continue
			} else if (cerr == io.EOF) || IsClosedError(cerr) {
				Logger.Debug("upstream: client reached end")
				return
				// first := !ct.isRunning()
				// if p.keep && ct.runContinue() {
				// 	if first {
				// 		Logger.Info("upstream: reconnect waiting")
				// 	}
				// 	sleep()
				// 	continue
				// } else {
				// 	return
				// }
			} else {
				Logger.ErrorE(cerr)
				return
			}
		}
		// ct.reset()
	}
}

func (p *Forwarder) downstream(clientConn, serverConn *ConnClosable, wg *sync.WaitGroup) {
	defer func() {
		Logger.Debug("downstream: finish")
		p.setDownstreamAlive(false)
		if wg != nil {
			wg.Done()
		}
	}()
	// ct := &CountdownTimer{Deadline: KeepSec}
	for p.getStreamsAlive() {
		var herr error
		if p.downBufUB == 0 {
			if serverConn.IsClosed() {
				return
			}
			if err := serverConn.SetReadDeadline(time.Now().Add(TimeoutDuration)); err != nil {
				Logger.Warn("Warning: serverConn.SetReadDeadline: " + err.Error())
			}
			p.downBufUB, herr = serverConn.Read(p.downBuf)
			p.downBufLB = 0
		}
		if p.downBufUB > 0 {
			for p.getStreamsAlive() {
				if clientConn.IsClosed() {
					return
				}
				if err := clientConn.SetWriteDeadline(time.Now().Add(TimeoutDuration)); err != nil {
					Logger.Warn("Warning: clientConn.SetWriteDeadline: " + err.Error())
				}
				m, cerr := clientConn.Write(p.downBuf[p.downBufLB:p.downBufUB])
				Logger.TraceF("downstream: Wrote: %s\n", p.downBuf[p.downBufLB:p.downBufUB])
				p.downBufUB += m
				if cerr != nil {
					if IsDeadlineExceeded(cerr) {
						continue
					} else if IsClosedError(cerr) {
						Logger.Debug("downstream: client is close")
						return
						// first := !ct.isRunning()
						// if p.keep && ct.runContinue() {
						// 	if first {
						// 		Logger.Info("downstream: reconnect waiting")
						// 	}
						// 	sleep()
						// 	continue
						// } else {
						// 	return
						// }
					} else {
						Logger.ErrorE(cerr)
						return
					}
				}
				p.downBufUB = 0
				p.downBufLB = 0
				break
			}
			// ct.reset()
		}
		if herr != nil {
			if IsDeadlineExceeded(herr) {
				continue
			}
			if herr == io.EOF {
				Logger.Debug("downstream: server reached end")
			} else {
				Logger.ErrorE(herr)
			}
			return
		}
	}
}

type net_TCPConn = net.TCPConn

type ConnClosable struct {
	*net_TCPConn
	cmux   sync.Mutex
	closed bool
}

func NewConnClosable(tcpConn *net.TCPConn) *ConnClosable {
	return &ConnClosable{
		net_TCPConn: tcpConn,
	}
}

func (p *ConnClosable) IsClosed() bool {
	defer p.cmux.Unlock()
	p.cmux.Lock()
	return p.closed
}

func (p *ConnClosable) Close() error {
	defer p.cmux.Unlock()
	p.cmux.Lock()
	if p.closed {
		return nil
	}
	err := p.net_TCPConn.Close()
	p.closed = true
	return err
}

type CountdownTimer struct {
	Deadline float64
	start    time.Time
	running  bool
}

func (p *CountdownTimer) reset() {
	p.running = false
}

func (p *CountdownTimer) isRunning() bool {
	return p.running
}

func (p *CountdownTimer) runContinue() bool {
	if !p.running {
		p.start = time.Now()
		p.running = true
	}
	return time.Now().Sub(p.start).Seconds() < p.Deadline
}

func sleep() {
	time.Sleep(RetryInterval)
}
