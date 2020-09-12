package main

/*
  TODO:
  - Session.Start() のロック内での Dial() や Close() は時間がかかりすぎるので回避する
  - Session.upstream() と downstream() が終了するとき、
    Session のメンバーではなく関数内でローカルの net.Conn をクローズする
  - ログのユーティリティクラスを作り、fmt.Println を置き換える
  - クライアントを二重にクローズする問題を解決する
*/

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
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
		return nil
	}
	head, err := ReadProtocolHeader(clientConn)
	if err != nil {
		clientConn.Close()
		return err
	}
	fmt.Println("Header: " + head.String())
	var hostAddr string
	isFwd := false
	fwdIp := head.DstIP
	fwdPort := head.DstPort
	if net.IPv4(0, 0, 0, 0).Equal(fwdIp) {
		if appAddr == "" {
			fmt.Fprintln(os.Stderr,
				"Fatal: 'only-forwarding' mode requires dstIP:dstPort on a header")
		}
		hostAddr = appAddr
	} else {
		if fwdPort == 0 {
			return errors.New(fmt.Sprintf("missing port in address (HostAddr=%s)", hostAddr))
		}
		hostAddr = fmt.Sprintf("%s:%d", fwdIp.String(), fwdPort)
		isFwd = true
	}
	if hostAddr == "" {
		return errors.New("HostAddr is empty")
	}
	key := SessionKey{head.SessionId, hostAddr}
	sesh, ok := p.ssss.LoadOrStore(key, &Session{keep: true})
	if !ok {
		fmt.Println("New session")
	} else {
		fmt.Println("Use existing session")
	}
	var headBytes []byte
	if isFwd {
		headBytes = head.Bytes()
	}
	go sesh.(*Session).Start(clientConn, network, hostAddr, head.Resume(), headBytes)
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
	headBytes []byte,
) {
	complete := false
	newHostConn := false
	defer func() {
		p.mux.Unlock()
		if newHostConn && len(headBytes) > 0 {
			p.hostConn.Write(headBytes)
		}
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
		} else {
			fmt.Println("Host open")
		}
		p.hostConn = conn
		p.hostOpen = true
		newHostConn = true
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
	if p.hostOpen {
		if err := p.hostConn.Close(); err != nil {
			PrintError(err)
		} else {
			fmt.Println("Host closed")
		}
		p.hostOpen = false
	}
	if p.clientOpen {
		if err := p.clientConn.Close(); err != nil {
			PrintError(err)
		} else {
			fmt.Println("Client closed")
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
			fmt.Printf("upstream: Read %dB: %s", n, buf[:n])
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
					} else {
						fmt.Println("Client closed")
					}
				}
				first := !ct.isRunning()
				if p.keep && ct.runContinue() {
					if first {
						fmt.Println("upstream: reconnect waiting")
					}
					sleep()
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
				m, cErr := clientConn.Write(buf[:n])
				if cErr == nil {
					fmt.Printf("downstream: Write %dB: %s\n", m, buf[:m])
					break
				} else {
					if IsClosedError(cErr) {
						first := !ct.isRunning()
						if p.keep && ct.runContinue() {
							if first {
								fmt.Println("downstream: reconnect waiting")
							}
							sleep()
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
		}
		if hErr != nil {
			if hErr != io.EOF && !IsClosedError(hErr) {
				PrintError(hErr)
			}
			return
		}
	}
}

func sleep() {
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
