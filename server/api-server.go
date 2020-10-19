package main

import (
	"encoding/json"
	"net"
	"time"
)

func StartAPIServer(chanClose chan interface{}) {
	var ln net.Listener
	ln, err := net.Listen("tcp", APIServerLocalAddr)
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
		go onConnection(conn)
	}
}

func onConnection(conn net.Conn) {
	defer func() {
		Logger.InfoF("Close: %v\n", conn.RemoteAddr())
		conn.Close()
	}()
	Logger.InfoF("Accept: %v\n", conn.RemoteAddr())
	conn.SetReadDeadline(time.Now().Add(timeout * time.Second))
	var bReq []byte
	bReq, err := Readline(conn)
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	Logger.Info("Request: " + string(bReq))
	var req Request
	if err := json.Unmarshal(bReq, &req); err != nil {
		Logger.ErrorE(err)
		return
	}
	switch req.Method {
	case "deploy":
		doDeployReq(&req)
	case "remove":
		doRemoveReq(&req)
	case "_checkpoint":

	default:
		doUnsupportedReq(&req)
	}
}

func doDeployReq(req *Request) {
	switch req.Deploy.Type {
	case DeployTypeNew:
		apiService.DeployNew(
			req.Deploy.Name,
			req.Deploy.NewApp.Image,
			NewPortMap(req.Deploy.NewApp.Port.In, req.Deploy.NewApp.Port.Ext),
			req.Deploy.NewApp.Env,
		)
	case DeployTypeFwd:
		apiService.DeployFwd(
			req.Deploy.Name,
			req.Deploy.Fwd.SrcAddr,
			NewPortMap(req.Deploy.Fwd.Port.In, req.Deploy.Fwd.Port.Ext),
		)
	default:
		Logger.Error("Unsupported deploy type: " + req.Deploy.Type)
	}
}

func doRemoveReq(req *Request) {
	name := req.Remove.Name
	apiService.Remove(name)
}

func doUnsupportedReq(req *Request) {
	Logger.Error("Unsupported request method: " + req.Method)
}
