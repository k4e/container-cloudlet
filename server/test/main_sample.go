package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Argument <dst> required")
		os.Exit(-1)
	}
	dst := os.Args[1]
	clientset, config, err := NewClient()
	if err != nil {
		panic(err)
	}
	kubePortFwd, err := NewKubePortForward(config, "default", "app-sample-pod", 31213, 8888,
		os.Stdout, os.Stderr)
	if err != nil {
		panic(err)
	}
	go func() {
		err = kubePortFwd.Start()
		if err != nil {
			panic(err)
		}
	}()
	if err := ExecutePod(clientset, config, "default", "app-sample-pod", "app-sample-c",
		nil, os.Stdout, os.Stderr, "/bin/sh", "-c", "echo hello"); err != nil {
		panic(err)
	}
	if err := ExecutePod(clientset, config, "default", "app-sample-pod", "app-sample-c",
		nil, os.Stdout, os.Stderr, "/bin/sh", "-c", "mkdir -p /tmp/testdir"); err != nil {
		panic(err)
	}
	if err := ExecutePod(clientset, config, "default", "app-sample-pod", "app-sample-c",
		nil, os.Stdout, os.Stderr, "/bin/sh", "-c", "echo 'Hello, World!!! 123' > /tmp/testdir/testfile"); err != nil {
		panic(err)
	}
	defer kubePortFwd.Close()
	select {
	case <-kubePortFwd.readyChan:
		break
	}
	fmt.Println("Port fowarding ready")
	if err := ExecutePod(clientset, config, "default", "app-sample-pod", "app-sample-c",
		nil, os.Stdout, os.Stderr, "/bin/sh", "-c", fmt.Sprintf("rsync -a /tmp/testdir rsync://%s", dst)); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	wg := sync.WaitGroup{}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	wg.Add(1)
	go func() {
		<-sigs
		fmt.Println("Bye...")
		wg.Done()
	}()
	wg.Wait()
}
