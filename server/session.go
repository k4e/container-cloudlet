package main

/*
  TODO:
  - Session.Start() のロック内での Dial() や Close() は時間がかかりすぎるので回避する
  - hostConn の保持によるコネクションの維持を実現する
*/

import (
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

const KeepSec = 60.
const TimeoutDuration = 1000 * time.Millisecond
const RetryInterval = 1000 * time.Millisecond
const BufferSize = 1024 * 1024

func IsDeadlineExceeded(err error) bool {
	nerr, ok := err.(net.Error)
	if !ok {
		return false
	}
	if !nerr.Timeout() {
		return false
	}
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		return false
	}
	return true
}

type SessionPool struct {
	seshs sync.Map
}

type SessionKey struct {
	sessionId uuid.UUID
	hostIP    string
	hostPort  int
}

func (p *SessionPool) Accept(
	ln *net.TCPListener,
	network string,
	hostAddr *net.TCPAddr,
	isExtHost bool,
) error {
	clientConn, err := ln.AcceptTCP()
	if err != nil {
		if !IsClosedError(err) {
			return err
		}
		return nil
	}
	Logger.Info("Accept: " + clientConn.RemoteAddr().String())
	Logger.Info("Default buffer size")
	head, err := ReadProtocolHeader(clientConn)
	if err != nil {
		return err
	}
	Logger.Info("Header: " + head.String())
	// var hostAddr* net.TCPAddr
	// isHostFwd := false
	// fwdIp := head.DstIP
	// fwdPort := head.DstPort
	// if net.IPv4(0, 0, 0, 0).Equal(fwdIp) {
	// 	if appAddr == nil {
	// 		Logger.Warn("Fatal: requires dstIP:dstPort on a header because the app isn't created")
	// 	}
	// 	hostAddr = appAddr
	// } else {
	// 	if fwdPort == 0 {
	// 		return errors.New(fmt.Sprintf("missing port in address (HostAddr=%s)", hostAddr))
	// 	}
	// 	hostAddrStr := fmt.Sprintf("%s:%d", fwdIp.String(), fwdPort)
	// 	var err error
	// 	hostAddr, err = net.ResolveTCPAddr(network, hostAddrStr)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	isHostFwd = true
	// }
	if hostAddr == nil {
		return errors.New("HostAddr is empty")
	}
	if err != nil {
		return err
	}
	key := SessionKey{head.SessionId, hostAddr.IP.String(), hostAddr.Port}
	sesh, ok := p.seshs.LoadOrStore(key, NewSession(true))
	if !ok {
		Logger.Info("New session")
	} else {
		Logger.Info("Use existing session")
	}
	var headBytes []byte
	if isExtHost {
		nextHead := ProtocolHeader{
			SessionId: head.SessionId,
			// DstIP:     net.IPv4(0, 0, 0, 0),
			// DstPort:   0,
			// Flag:      head.Flag,
		}
		headBytes = nextHead.Bytes()
	}
	go sesh.(*Session).Start(NewConnection(clientConn), network, hostAddr, false, headBytes)
	return nil
}

type Session struct {
	keep      bool
	mux       sync.Mutex
	hostConn  *Connection
	muxStream sync.Mutex
	upAlive   bool
	upBuf     []byte
	upBufLB   int
	upBufUB   int
	downAlive bool
	downBuf   []byte
	downBufLB int
	downBufUB int
}

func NewSession(keep bool) *Session {
	p := &Session{keep: true}
	p.upBuf = make([]byte, BufferSize)
	p.downBuf = make([]byte, BufferSize)
	return p
}

func (p *Session) Start(
	clientConn *Connection,
	network string,
	hostAddr *net.TCPAddr,
	resume bool,
	headBytes []byte,
) {
	p.setStreamsAlive(false)
	defer func() {
		p.mux.Unlock()
	}()
	p.mux.Lock()
	wg := &sync.WaitGroup{}
	var hostConn *Connection
	if true {
		conn, err := net.DialTCP(network, nil, hostAddr)
		if err != nil {
			Logger.ErrorE(err)
			Logger.Info("Client close")
			if err := clientConn.Close(); err != nil {
				Logger.Warn("Warning: clientConn.Close: " + err.Error())
			}
			return
		}
		Logger.Info("Host open")
		hostConn = NewConnection(conn)
		if len(headBytes) > 0 {
			hostConn.Write(headBytes)
		}
	}
	p.setStreamsAlive(true)
	wg.Add(2)
	go p.upstream(clientConn, hostConn, wg)
	go p.downstream(clientConn, hostConn, wg)
	wg.Wait()
	clientConn.Close()
	hostConn.Close()
}

func (p *Session) getStreamsAlive() bool {
	defer p.muxStream.Unlock()
	p.muxStream.Lock()
	return p.upAlive && p.downAlive
}

func (p *Session) setStreamsAlive(b bool) {
	defer p.muxStream.Unlock()
	p.muxStream.Lock()
	p.upAlive = b
	p.downAlive = b
}

func (p *Session) setUpstreamAlive(b bool) {
	defer p.muxStream.Unlock()
	p.muxStream.Lock()
	p.upAlive = b
}

func (p *Session) setDownstreamAlive(b bool) {
	defer p.muxStream.Unlock()
	p.muxStream.Lock()
	p.downAlive = b
}

func (p *Session) upstream(clientConn, hostConn *Connection, wg *sync.WaitGroup) {
	defer func() {
		Logger.Info("upstream: finish")
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
				if hostConn.IsClosed() {
					return
				}
				if err := hostConn.SetWriteDeadline(time.Now().Add(TimeoutDuration)); err != nil {
					Logger.Warn("Warning: hostConn.SetWriteDeadline: " + err.Error())
				}
				m, herr := hostConn.Write(p.upBuf[p.upBufLB:p.upBufUB])
				Logger.DebugF("upstream: Wrote: %s", p.upBuf[p.upBufLB:p.upBufUB])
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
				Logger.Info("upstream: client reached end")
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

func (p *Session) downstream(clientConn, hostConn *Connection, wg *sync.WaitGroup) {
	defer func() {
		Logger.Info("downstream: finish")
		p.setDownstreamAlive(false)
		if wg != nil {
			wg.Done()
		}
	}()
	// ct := &CountdownTimer{Deadline: KeepSec}
	for p.getStreamsAlive() {
		var herr error
		if p.downBufUB == 0 {
			if hostConn.IsClosed() {
				return
			}
			if err := hostConn.SetReadDeadline(time.Now().Add(TimeoutDuration)); err != nil {
				Logger.Warn("Warning: hostConn.SetReadDeadline: " + err.Error())
			}
			p.downBufUB, herr = hostConn.Read(p.downBuf)
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
				Logger.DebugF("downstream: Wrote: %s\n", p.downBuf[p.downBufLB:p.downBufUB])
				p.downBufUB += m
				if cerr != nil {
					if IsDeadlineExceeded(cerr) {
						continue
					} else if IsClosedError(cerr) {
						Logger.Info("downstream: client is close")
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
				Logger.Info("downstream: host reached end")
			} else {
				Logger.ErrorE(herr)
			}
			return
		}
	}
}

type net_TCPConn = net.TCPConn

type Connection struct {
	*net_TCPConn
	cmux   sync.Mutex
	closed bool
}

func NewConnection(tcpConn *net.TCPConn) *Connection {
	return &Connection{
		net_TCPConn: tcpConn,
	}
}

func (p *Connection) IsClosed() bool {
	defer p.cmux.Unlock()
	p.cmux.Lock()
	return p.closed
}

func (p *Connection) Close() error {
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
