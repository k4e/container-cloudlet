package main

/*
  TODO:
  - Session.Start() のロック内での Dial() や Close() は時間がかかりすぎるので回避する
  - Session.upstream() と downstream() が終了するとき、
    Session のメンバーではなく関数内でローカルの net.Conn をクローズする
  - ログのユーティリティクラスを作り、fmt.Println を置き換える
*/

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const KeepSec = 60.
const RetryInterval = 1000 * time.Millisecond
const BufSize = 64 * 1024

func IsClosedError(e error) bool {
	msg := "use of closed network connection"
	return strings.Contains(e.Error(), msg)
}

type SessionPool struct {
	ssss sync.Map
}

type SessionKey struct {
	sessionId uuid.UUID
	hostAddr  string
}

func (p *SessionPool) Accept(ln net.Listener, network, appAddr string) error {
	clientConn, err := ln.Accept()
	if err != nil {
		if !IsClosedError(err) {
			return err
		}
	}
	head, err := ReadProtocolHeader(clientConn)
	if err != nil {
		clientConn.Close()
		return err
	}
	fmt.Println("Header: " + head.String())
	var hostAddr string
	if net.IPv4(0, 0, 0, 0).Equal(head.hostIP) {
		hostAddr = appAddr
	} else {
		hostAddr = head.hostIP.String()
	}
	key := SessionKey{head.sessionId, hostAddr}
	sesh, ok := p.ssss.LoadOrStore(key, &Session{keep: true})
	if !ok {
		fmt.Println("New session")
	} else {
		fmt.Println("Use existing session")
	}
	go sesh.(*Session).Start(clientConn, network, hostAddr, head.Resume())
	return nil
}

type Session struct {
	keep       bool
	mux        sync.Mutex
	clientConn net.Conn
	hostConn   net.Conn
	clientOpen bool
	hostOpen   bool
	streaming  bool
	muxStream  sync.Mutex
}

func (p *Session) Start(
	clientConn net.Conn,
	network string,
	hostAddr string,
	resume bool,
) {
	complete := false
	defer func() {
		p.mux.Unlock()
		if complete {
			go p.stream()
		}
	}()
	p.mux.Lock()
	if !p.hostOpen {
		conn, err := net.Dial(network, hostAddr)
		if err != nil {
			PrintError(err)
			return
		}
		p.hostConn = conn
		p.hostOpen = true
	}
	if p.clientOpen {
		if err := p.clientConn.Close(); err != nil {
			PrintError(err)
		}
		p.clientOpen = false
	}
	p.clientConn = clientConn
	p.clientOpen = true
	complete = true
}

func (p *Session) Close() {
	defer p.mux.Unlock()
	p.mux.Lock()
	if p.clientOpen {
		if err := p.clientConn.Close(); err != nil {
			PrintError(err)
		}
		p.clientOpen = false
	}
	if p.clientOpen {
		if err := p.clientConn.Close(); err != nil {
			PrintError(err)
		}
		p.clientOpen = false
	}
}

func (p *Session) IsClosed() bool {
	defer p.mux.Unlock()
	p.mux.Lock()
	return !p.clientOpen && !p.hostOpen
}

func (p *Session) getClientConn() (net.Conn, bool) {
	defer p.mux.Unlock()
	p.mux.Lock()
	return p.clientConn, p.clientOpen
}

func (p *Session) getHostConn() (net.Conn, bool) {
	defer p.mux.Unlock()
	p.mux.Lock()
	return p.hostConn, p.hostOpen
}

func (p *Session) stream() {
	defer p.muxStream.Unlock()
	p.muxStream.Lock()
	if !p.streaming {
		fmt.Println("New streaming")
		go p.upstream()
		go p.downstream()
		p.streaming = true
	} else {
		fmt.Println("Use existing streaming")
	}
}

func (p *Session) finishStream() {
	p.muxStream.Lock()
	p.streaming = false
	p.muxStream.Unlock()
	p.Close()
}

func (p *Session) upstream() {
	defer func() {
		fmt.Println("Finish upstream")
		p.finishStream()
	}()
	buf := make([]byte, BufSize)
	ct := &CountdownTimer{Deadline: KeepSec}
	for {
		clientConn, _ := p.getClientConn()
		n, cErr := clientConn.Read(buf)
		if n > 0 {
			hostConn, _ := p.getHostConn()
			m, hErr := hostConn.Write(buf[:n])
			if n != m {
				PrintErrorS(fmt.Sprintf("upstream: Read: %dB, Write: %dB", n, m))
			}
			if hErr != nil {
				if !IsClosedError(hErr) {
					PrintError(hErr)
				}
				return
			}
		}
		if cErr != nil {
			eof := cErr == io.EOF
			if eof || IsClosedError(cErr) {
				if eof {
					if err := clientConn.Close(); err != nil {
						PrintError(err)
					}
				}
				first := !ct.isRunning()
				if p.keep && ct.runContinue() {
					if first {
						fmt.Println("upstream: reconnect waiting")
					}
					p.sleep()
					continue
				} else {
					return
				}
			} else {
				PrintError(cErr)
				return
			}
		}
		ct.reset()
	}
}

func (p *Session) downstream() {
	defer func() {
		fmt.Println("Finish downstream")
		p.finishStream()
	}()
	buf := make([]byte, BufSize)
	ct := &CountdownTimer{Deadline: KeepSec}
	for {
		hostConn, _ := p.getHostConn()
		n, hErr := hostConn.Read(buf)
		if n > 0 {
			for {
				clientConn, _ := p.getClientConn()
				_, cErr := clientConn.Write(buf[:n])
				if cErr == nil {
					break
				} else {
					if IsClosedError(cErr) {
						first := !ct.isRunning()
						if p.keep && ct.runContinue() {
							if first {
								fmt.Println("downstream: reconnect waiting")
							}
							p.sleep()
							continue
						} else {
							return
						}
					} else {
						PrintError(cErr)
						return
					}
				}
			}
			ct.reset()
			fmt.Println("Wrote to client")
		}
		if hErr != nil {
			if hErr != io.EOF && !IsClosedError(hErr) {
				PrintError(hErr)
			}
			return
		}
	}
}

func (p *Session) sleep() {
	time.Sleep(RetryInterval)
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
