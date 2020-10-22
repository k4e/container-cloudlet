package main

import (
	"encoding/json"
	"net"
	"time"
)

func StartAPIServer(addr string, chanClose chan interface{}) {
	var ln net.Listener
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	open := true
	go func() {
		<-chanClose
		open = false
		ln.Close()
	}()
	for open {
		conn, err := ln.Accept()
		if err != nil {
			if IsClosedError(err) {
				Logger.Info("API server close")
				return
			} else {
				Logger.ErrorE(err)
				continue
			}
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func() {
		Logger.InfoF("Close: %v\n", conn.RemoteAddr())
		conn.Close()
	}()
	Logger.InfoF("Accept: %v\n", conn.RemoteAddr())
	conn.SetReadDeadline(time.Now().Add(RequestTimeout * time.Second))
	b, err := Readline(conn)
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	Logger.Info("Request: " + string(b))
	var req Request
	if err := json.Unmarshal(b, &req); err != nil {
		Logger.ErrorE(err)
		return
	}
	var resp *Response
	switch req.Method {
	case "deploy":
		doDeployReq(&req)
	case "remove":
		doRemoveReq(&req)
	case "_checkpoint":
		resp = doCheckpointReq(&req)
	default:
		doUnsupportedReq(&req)
	}
	if resp != nil {
		b, err := json.Marshal(resp)
		if err != nil {
			Logger.ErrorE(err)
			return
		}
		_, err = conn.Write(b)
		if err != nil {
			Logger.ErrorE(err)
			return
		}
	}
}

func doDeployReq(req *Request) {
	TheAPICore.Deploy(req)
}

func doRemoveReq(req *Request) {
	TheAPICore.Remove(req)
}

func doCheckpointReq(req *Request) *Response {
	return TheAPICore.Checkpoint(req)
}

func doUnsupportedReq(req *Request) {
	Logger.Error("Unsupported request method: " + req.Method)
}
