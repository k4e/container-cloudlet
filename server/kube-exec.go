package main

import (
	"io"
	"net/http"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func ExecutePod(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace string,
	podName string,
	containerName string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	commands ...string,
) error {
	cmd := commands
	opt := &apiv1.PodExecOptions{
		Command: cmd,
		Stdin:   (stdin != nil),
		Stdout:  (stdout != nil),
		Stderr:  (stderr != nil),
		TTY:     false,
	}
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(namespace).SubResource("exec").Param("container", containerName)
	req.VersionedParams(opt, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(config, http.MethodPost, req.URL())
	if err != nil {
		return err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return err
	}
	return nil
}
