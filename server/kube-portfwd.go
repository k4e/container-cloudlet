package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport},
		http.MethodPost, &url)
	portForwarder, err := portforward.NewOnAddresses(dialer, addrs, ports,
		stopChan, readyChan, out, errOut)
	return portForwarder, err
}
