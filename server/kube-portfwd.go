package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func NewKubePortForward(
	config *rest.Config,
	namespace string,
	podName string,
	localPort int,
	podPort int,
	out io.Writer,
	errOut io.Writer,
	stopChan chan struct{},
	readyChan chan struct{},
) (*portforward.PortForwarder, error) {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	host := strings.TrimLeft(config.Host, "htps:/")
	url := url.URL{Scheme: "https", Path: path, Host: host}
	addrs := []string{"0.0.0.0"}
	ports := []string{fmt.Sprintf("%d:%d", localPort, podPort)}
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport},
		http.MethodPost, &url)
	portForwarder, err := portforward.NewOnAddresses(dialer, addrs, ports,
		stopChan, readyChan, out, errOut)
	return portForwarder, errors.WithStack(err)
}

func OpenKubePortForwardReady(
	config *rest.Config,
	namespace string,
	podName string,
	localPort int,
	podPort int,
	out io.Writer,
	errOut io.Writer,
	stopChan chan struct{},
) error {
	readyChan := make(chan struct{}, 1)
	errorChan := make(chan error, 1)
	kubePortFwd, err := NewKubePortForward(config, namespace, podName, localPort, podPort,
		out, errOut, stopChan, readyChan)
	if err != nil {
		return err
	}
	go func() {
		if err := kubePortFwd.ForwardPorts(); err != nil {
			Logger.Warn(err.Error())
			errorChan <- err
		}
	}()
	select {
	case <-readyChan:
	case err := <-errorChan:
		return err
	}
	return nil
}
