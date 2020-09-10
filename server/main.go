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
	fmt.Println("Interface IP addresses ...")
	PrintInterfaceAddrs()

	ln, err := net.Listen("tcp", local_addr)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func() {
		fmt.Printf("Close: %v\n", conn.RemoteAddr())
		conn.Close()
	}()
	fmt.Printf("Accept: %v\n", conn.RemoteAddr())
	conn.SetReadDeadline(time.Now().Add(timeout * time.Second))
	var bReq []byte
	bReq, err := Readline(conn)
	if err != nil {
		PrintError(err)
		return
	}
	fmt.Println("Request: " + string(bReq))
	var req Request
	if err := json.Unmarshal(bReq, &req); err != nil {
		PrintError(err)
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
	podName := req.Create.Name + "-pod"
	containerName := req.Create.Name + "-c"
	port := int32(req.Create.Port)
	extPort := int32(req.Create.ExtPort)
	env := req.Create.Env
	serviceName := req.Create.Name + "-svc"
	clusterIpName := req.Create.Name + "-cip"
	clientset, err := NewClient()
	if err != nil {
		PrintError(err)
		return
	}
	if pod, err := CreatePod(
		clientset,
		podName,
		name,
		containerName,
		req.Create.Image,
		port,
		env,
	); err == nil {
		fmt.Println("Created pod: " + pod.GetName())
	} else {
		PrintError(err)
	}
	clusterIP := ""
	if svc, err := GetService(clientset, serviceName); err == nil && svc.Spec.ClusterIP != "" {
		fmt.Println("Use existing service: " + svc.GetName())
		clusterIP = svc.Spec.ClusterIP
	} else if svc, err := CreateService(
		clientset,
		serviceName,
		name,
		clusterIpName,
		port,
	); err == nil {
		fmt.Println("Created service: " + svc.GetName())
		clusterIP = svc.Spec.ClusterIP
	} else {
		PrintError(err)
	}
	if _, ok := fwdsvcs.Load(name); ok {
		fmt.Println("Use existing forwarding service")
	} else if clusterIP != "" {
		clientAddr := fmt.Sprintf(":%d", extPort)
		appAddr := fmt.Sprintf("%s:%d", clusterIP, port)
		if f, err := StartForwardingService("tcp", clientAddr, appAddr); err == nil {
			fwdsvcs.Store(name, f)
		} else {
			PrintError(err)
		}
	} else {
		PrintErrorS("Error: forwarding service cannot not started because ClusterIP is unknown")
	}
}

func doDelete(req *Request) {
	name := req.Delete.Name
	podName := req.Delete.Name + "-pod"
	serviceName := req.Delete.Name + "-svc"
	clientset, err := NewClient()
	if err != nil {
		PrintError(err)
		return
	}
	if v, ok := fwdsvcs.Load(name); ok {
		if err := v.(*ForwardingService).Close(); err != nil {
			PrintError(err)
		}
		fwdsvcs.Delete(name)
	}
	if err := DeleteService(clientset, serviceName); err == nil {
		fmt.Println("Deleted service: " + serviceName)
	} else {
		PrintError(err)
	}
	if err := DeletePod(clientset, podName); err == nil {
		fmt.Println("Deleted pod: " + podName)
	} else {
		PrintError(err)
	}
}

func doUnsupported(req *Request) {
	PrintErrorS("Unsupported operation")
}

func PrintError(e error) {
	fmt.Fprintf(os.Stderr, "%v\n", e)
}

func PrintErrorS(s string) {
	fmt.Fprintln(os.Stderr, s)
}
