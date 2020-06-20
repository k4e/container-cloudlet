package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

const local_addr = ":9999"
const timeout = 30

var fwdsvc = make(map[string](*ForwardingService))

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
	nodePort := int32(req.Create.NodePort)
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
	); err == nil {
		fmt.Println("Created pod: " + pod.GetName())
	} else {
		PrintError(err)
	}
	if svc, err := CreateService(
		clientset,
		serviceName,
		name,
		clusterIpName,
		port,
	); err == nil {
		fmt.Println("Created service: " + svc.GetName())
		clientAddr := fmt.Sprintf(":%d", nodePort)
		hostAddr := fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, port)
		if f, err := NewForwarding("tcp", clientAddr, hostAddr); err == nil {
			fwdsvc[name] = f
			if err := f.Start(); err != nil {
				PrintError(err)
			}
		} else {
			PrintError(err)
		}
	} else {
		PrintError(err)
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
	if f, ok := fwdsvc[name]; ok {
		if err := f.Close(); err != nil {
			PrintError(err)
		}
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
	PrintErrorS(e.Error())
}

func PrintErrorS(s string) {
	fmt.Fprintln(os.Stderr, s)
}
