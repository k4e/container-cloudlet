package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	APIServerPort  = 9999
	RequestTimeout = 30
)

var TheAPICore *APICore

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
	hostConf, err := LoadHostConf()
	if err != nil {
		panic(err)
	}
	hostAddr, err := GetInterfaceAddr(hostConf.HostNetworkInterface)
	if err != nil {
		panic(err)
	}
	fmt.Printf("This host network address: %s (%s)\n", hostAddr, hostConf.HostNetworkInterface)
	TheAPICore = NewAPICore(hostConf, hostAddr)
	fmt.Println("Interface IP addresses:")
	if err := PrintInterfaceAddrs("- "); err != nil {
		panic(err)
	}
	apiServerAddr := fmt.Sprintf(":%d", APIServerPort)
	chanClose := make(chan interface{})
	go StartAPIServer(apiServerAddr, chanClose)
	fmt.Println("API server is starting at: " + apiServerAddr)
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
		case "tunnel":
			doTunnelCmd(args)
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

func doTunnelCmd(args []string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <local-addr> <remote-addr>\n", args[0])
		return
	}
	localAddr := args[1]
	remoteAddr := args[2]
	sshClient, err := NewSSHClient(TheAPICore.HostConf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	chanClose := make(chan struct{})
	sshClient.OpenTunnel(localAddr, remoteAddr, chanClose)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Open SSH tunnel; Ctrl-C to abort")
	<-sigs
	signal.Reset(syscall.SIGINT, syscall.SIGTERM)
	close(chanClose)
}

func doUnsupportedCmd(args []string) {
	fmt.Println("Unsupported command: " + args[0])
}
