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
			fmt.Fprintf(os.Stderr, "%v\n", err)
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
		printError(err)
		return
	}
	fmt.Println("Request: " + string(bReq))
	var req Request
	if err := json.Unmarshal(bReq, &req); err != nil {
		printError(err)
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
	label := req.Create.Name
	podName := req.Create.Name + "-pod"
	containerName := req.Create.Name + "-c"
	serviceName := req.Create.Name + "-svc"
	nodePortName := req.Create.Name + "-np"
	port := int32(req.Create.Port)
	nodePort := int32(req.Create.NodePort)
	clientset, err := NewClient()
	if err != nil {
		printError(err)
		return
	}
	if pod, err := CreatePod(
		clientset,
		podName,
		label,
		containerName,
		req.Create.Image,
		port,
	); err == nil {
		fmt.Println("Created pod: " + pod.GetName())
	} else {
		printError(err)
	}
	if svc, err := CreateService(
		clientset,
		serviceName,
		label,
		nodePortName,
		port,
		nodePort,
	); err == nil {
		fmt.Println("Created service: " + svc.GetName())
	} else {
		printError(err)
	}
}

func doDelete(req *Request) {
	podName := req.Delete.Name + "-pod"
	serviceName := req.Delete.Name + "-svc"
	clientset, err := NewClient()
	if err != nil {
		printError(err)
		return
	}
	if err := DeletePod(clientset, podName); err == nil {
		fmt.Println("Deleted pod: " + podName)
	} else {
		printError(err)
	}
	if err := DeleteService(clientset, serviceName); err == nil {
		fmt.Println("Deleted service: " + serviceName)
	} else {
		printError(err)
	}
}

func doUnsupported(req *Request) {
	printErrorS("Unsupported operation")
}

func printError(e error) {
	printErrorS(e.Error())
}

func printErrorS(s string) {
	fmt.Fprintln(os.Stderr, s)
}
