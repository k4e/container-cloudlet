package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"

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
		return errors.WithStack(err)
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func GetPodEnv(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace string,
	podName string,
	containerName string,
	stderr io.Writer,
	env string,
) (string, error) {
	stdout := &bytes.Buffer{}
	cmd := []string{"/bin/sh", "-c", fmt.Sprintf("echo ${%s}", env)}
	if err := ExecutePod(clientset, config, namespace, podName, containerName,
		nil, stdout, stderr, cmd...); err != nil {
		return "", err
	}
	v := stdout.String()
	return v, nil
}

func SetPodEnv(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace string,
	podName string,
	containerName string,
	stderr io.Writer,
	env string,
	value string,
) error {
	cmd := []string{"/bin/sh", "-c", fmt.Sprintf("export %s=%s", env, value)}
	if err := ExecutePod(clientset, config, namespace, podName, containerName,
		nil, nil, stderr, cmd...); err != nil {
		return err
	}
	return nil
}

func ReadPodFile(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace string,
	podName string,
	containerName string,
	stderr io.Writer,
	path string,
) (string, error) {
	stdout := &bytes.Buffer{}
	cmd := []string{"/bin/sh", "-c", fmt.Sprintf("cat %s", path)}
	if err := ExecutePod(clientset, config, namespace, podName, containerName,
		nil, stdout, stderr, cmd...); err != nil {
		return "", err
	}
	v := stdout.String()
	return v, nil
}

func WritePodFile(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace string,
	podName string,
	containerName string,
	stderr io.Writer,
	path string,
	content string,
	permission string,
) error {
	stdin := &bytes.Buffer{}
	stdin.WriteString(content)
	chmod := ""
	if permission != "" {
		chmod = fmt.Sprintf("&& chmod %s %s", permission, path)
	}
	cmd := []string{"/bin/sh", "-c", fmt.Sprintf("cat > %s %s", path, chmod)}
	if err := ExecutePod(clientset, config, namespace, podName, containerName,
		stdin, nil, stderr, cmd...); err != nil {
		return err
	}
	return nil
}
