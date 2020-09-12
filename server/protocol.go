package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/google/uuid"
)

type ProtocolHeader struct {
	SessionId uuid.UUID
	DstIP     net.IP
	DstPort   uint16
	Flag      byte
}

func ReadProtocolHeader(conn net.Conn) (*ProtocolHeader, error) {
	var head ProtocolHeader
	buf := make([]byte, 23)
	if n, err := conn.Read(buf); err != nil {
		return nil, err
	} else if n < 23 {
		return nil, errors.New(fmt.Sprintf("Insufficient session protocol header: %d byte", n))
	}
	var sessionId uuid.UUID
	sessionId, err := uuid.FromBytes(buf[0:16])
	if err != nil {
		return nil, err
	}
	dstIP := net.IPv4(buf[16], buf[17], buf[18], buf[19])
	dstPort := binary.BigEndian.Uint16(buf[20:22])
	flag := buf[22]
	head.SessionId = sessionId
	head.DstIP = dstIP
	head.DstPort = dstPort
	head.Flag = flag
	return &head, nil
}

func (p *ProtocolHeader) Resume() bool {
	return p.Flag&0x1 != 0
}

func (p *ProtocolHeader) Bytes() []byte {
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, p.DstPort)
	var ans []byte
	ans = append(ans, p.SessionId[:]...)
	ans = append(ans, p.DstIP.To4()[0:4]...)
	ans = append(ans, portBytes...)
	ans = append(ans, p.Flag)
	return ans
}

func (p *ProtocolHeader) String() string {
	return fmt.Sprintf("sessionId=%s, hostIP=%s, port=%d, flag=0x%02x",
		p.SessionId.String(), p.DstIP.String(), p.DstPort, p.Flag)
}
