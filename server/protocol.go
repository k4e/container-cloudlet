package main

import (
	"errors"
	"fmt"
	"net"

	"github.com/google/uuid"
)

type ProtocolHeader struct {
	sessionId uuid.UUID
	hostIP    net.IP
	flag      byte
}

func ReadProtocolHeader(conn net.Conn) (*ProtocolHeader, error) {
	var head ProtocolHeader
	buf := make([]byte, 21)
	if n, err := conn.Read(buf); err != nil {
		return nil, err
	} else if n < 21 {
		return nil, errors.New(fmt.Sprintf("Insufficient session protocol header: %d byte", n))
	}
	var sessionId uuid.UUID
	sessionId, err := uuid.FromBytes(buf[0:16])
	if err != nil {
		return nil, err
	}
	hostIP := net.IPv4(buf[16], buf[17], buf[18], buf[19])
	flag := buf[20]
	head.sessionId = sessionId
	head.hostIP = hostIP
	head.flag = flag
	return &head, nil
}

func (p *ProtocolHeader) Resume() bool {
	return p.flag & 0x1 != 0
}

func (p *ProtocolHeader) String() string {
	return fmt.Sprintf("sessionId=%s, hostIP=%s, flag=0x%02x",
		p.sessionId.String(), p.hostIP.String(), p.flag)
}
