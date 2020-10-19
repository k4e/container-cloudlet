package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const APIServerLocalAddr = ":9999"
const timeout = 30

var apiService *APIService

func main() {
	interactive := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-v":
			Logger.SetVerbosity(true)
		case "-i":
			interactive = true
		default:
			Logger.Warn("Warning: ignored arg: " + arg)
		}
	}
	apiService = NewAPIService()
	fmt.Println("Interface IP addresses:")
	PrintInterfaceAddrs("- ")
	chanClose := make(chan interface{})
	go startAPIServer(chanClose)
	fmt.Println("API server is starting at: " + APIServerLocalAddr)
	if interactive {
		startCommandLine()
	} else {
		waitForSignal()
	}
	close(chanClose)
	time.Sleep(time.Millisecond * 100)
	fmt.Println("Bye")
}

func startCommandLine() {
	scan := bufio.NewScanner(os.Stdin)
	for {

		fmt.Print("> ")
		scan.Scan()
		line := scan.Text()
		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}
		switch args[0] {
		case "/q":
			return
		case "quit":
			return
		default:
			doUnsupportedCmd(args)
		}
	}
}

func waitForSignal() {
	wg := sync.WaitGroup{}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	wg.Add(1)
	go func() {
		<-sigs
		wg.Done()
	}()
	wg.Wait()
}

func startAPIServer(chanClose chan interface{}) {
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

func argsToMap(args []string) map[string]string {
	m := map[string]string{}
	for _, v := range args {
		a := strings.SplitN(v, "=", 2)
		if len(a) == 1 {
			m[a[0]] = ""
		} else if len(a) == 2 {
			m[a[0]] = a[1]
		} else {
			fmt.Fprintf(os.Stderr, "Warning: ignored arg: "+v)
		}
	}
	return m
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

func doUnsupportedCmd(args []string) {
	fmt.Println("Unsupported command: " + args[0])
}

func doUnsupportedReq(req *Request) {
	Logger.Error("Unsupported request method: " + req.Method)
}
