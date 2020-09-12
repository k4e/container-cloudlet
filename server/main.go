package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

const local_addr = ":9999"
const timeout = 30

var fwdsvcs sync.Map

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "-v" {
			Logger.SetVerbosity(true)
		}
	}
	fmt.Println("Interface IP addresses ...")
	PrintInterfaceAddrs()
	ln, err := net.Listen("tcp", local_addr)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			Logger.ErrorE(err)
			continue
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
	switch req.Op {
	case "create":
		doCreate(&req)
	case "delete":
		doDelete(&req)
	default:
		doUnsupported(&req)
	}
}

func doCreate(req *Request) {
	name := req.Create.Name
	image := req.Create.Image
	podName := req.Create.Name + "-pod"
	containerName := req.Create.Name + "-c"
	port := int32(req.Create.Port)
	extPort := int32(req.Create.ExtPort)
	env := req.Create.Env
	serviceName := req.Create.Name + "-svc"
	clusterIpName := req.Create.Name + "-cip"
	clientset, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	clusterIP := ""
	if req.Create.CreateApp {
		if pod, err := CreatePod(
			clientset,
			podName,
			name,
			containerName,
			image,
			port,
			env,
		); err == nil {
			Logger.Info("Created pod: " + pod.GetName())
		} else {
			Logger.ErrorE(err)
		}
		if svc, err := GetService(clientset, serviceName); err == nil && svc.Spec.ClusterIP != "" {
			Logger.Info("Use existing service: " + svc.GetName())
			clusterIP = svc.Spec.ClusterIP
		} else if svc, err := CreateService(
			clientset,
			serviceName,
			name,
			clusterIpName,
			port,
		); err == nil {
			Logger.Info("Created service: " + svc.GetName())
			clusterIP = svc.Spec.ClusterIP
		} else {
			Logger.ErrorE(err)
		}
	}
	if _, ok := fwdsvcs.Load(name); ok {
		Logger.Info("Use existing forwarding service")
	} else if !req.Create.CreateApp || clusterIP != "" {
		clientAddr := fmt.Sprintf(":%d", extPort)
		var appAddr string
		if clusterIP != "" {
			appAddr = fmt.Sprintf("%s:%d", clusterIP, port)
		}
		if f, err := StartForwardingService("tcp", clientAddr, appAddr); err == nil {
			fwdsvcs.Store(name, f)
		} else {
			Logger.ErrorE(err)
		}
	} else {
		Logger.Error("Error: forwarding service cannot not started because ClusterIP is unknown")
	}
}

func doDelete(req *Request) {
	name := req.Delete.Name
	podName := req.Delete.Name + "-pod"
	serviceName := req.Delete.Name + "-svc"
	clientset, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	if v, ok := fwdsvcs.Load(name); ok {
		if err := v.(*ForwardingService).Close(); err != nil {
			Logger.ErrorE(err)
		}
		fwdsvcs.Delete(name)
	}
	if err := DeleteService(clientset, serviceName); err == nil {
		Logger.Info("Deleted service: " + serviceName)
	} else {
		Logger.ErrorE(err)
	}
	if err := DeletePod(clientset, podName); err == nil {
		Logger.Info("Deleted pod: " + podName)
	} else {
		Logger.ErrorE(err)
	}
}

func doUnsupported(req *Request) {
	Logger.Error("Unsupported operation")
}
