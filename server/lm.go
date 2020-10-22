package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	LMRsyncModuleName        = "tmp"
	LMRsyncModuleDirectory   = "/tmp"
	LMCheckpointImagesDir    = "/tmp/cloudlet.lm/images"
	HostLiveMigrationPort    = 19999
	PodCheckpointRestorePort = 19999
	PodRsyncPort             = 873
	MainPidFilePath          = "/MAIN_PID"
)

func GetRestorePodCommand() ([]string, []string) {
	return []string{"/bin/sh", "-c", "--"}, []string{"while true; do sleep 60; done"}
}

type LM_Restore struct {
	HostConf         *HostConf
	Clientset        kubernetes.Interface
	RestConfig       *rest.Config
	ThisAddr         string
	DstNamespace     string
	DstPodName       string
	DstContainerName string
	SrcAddr          string
	SrcAPIServerAddr string
	SrcName          string
}

func (p *LM_Restore) Exec() error {
	k8sPortFwdCloseChan := make(chan struct{})
	closeK8sPortFwd := func() {
		if k8sPortFwdCloseChan != nil {
			close(k8sPortFwdCloseChan)
			k8sPortFwdCloseChan = nil
		}
	}
	defer closeK8sPortFwd()
	Logger.Debug("[Restore] open kube port-forward")
	if err := OpenKubePortForwardReady(p.RestConfig, p.DstNamespace, p.DstPodName,
		HostLiveMigrationPort, PodRsyncPort, os.Stdout, os.Stderr, k8sPortFwdCloseChan); err != nil {
		return err
	}
	Logger.Debug("[Restore] exec mkdir")
	if err := ExecutePod(p.Clientset, p.RestConfig, p.DstNamespace, p.DstPodName, p.DstContainerName, nil, os.Stdout, os.Stderr,
		"/bin/sh", "-c", fmt.Sprintf("mkdir -p %s", LMCheckpointImagesDir)); err != nil {
		return err
	}
	Logger.Debug("[Restore] exec rsync --daemon")
	if err := ExecutePod(p.Clientset, p.RestConfig, p.DstNamespace, p.DstPodName, p.DstContainerName, nil, os.Stdout, os.Stderr,
		"/bin/sh", "-c", "rsync --daemon"); err != nil {
		return err
	}
	Logger.Debug("[Restore] send checkpoint request")
	if resp, err := p.sendCheckpointRequest(); err != nil {
		return err
	} else if !resp.Ok {
		return errors.New("Checkpoint response error: " + resp.Msg)
	}
	// if err := ExecutePod(p.Clientset, p.RestConfig, p.DstNamespace, p.DstPodName, p.DstContainerName, nil, &p.log, &p.log,
	// 	"/bin/sh", "-c", fmt.Sprintf(
	// 		"criu lazy-pages --images-dir %s --page-server --address %s --port %d >%s/lazy-pages.log 2>&1 &",
	// 		LMCheckpointImagesDir, p.ThisAddr, HostLiveMigrationPort, LMLogDir)); err != nil {
	// 	return err
	// }
	Logger.Debug("[Restore] exec unshare criu restore")
	if err := ExecutePod(p.Clientset, p.RestConfig, p.DstNamespace, p.DstPodName, p.DstContainerName, nil, os.Stdout, os.Stderr,
		"/bin/sh", "-c", fmt.Sprintf(
			"unshare -p -m --fork --mount-proc criu restore --images-dir %s --tcp-established --shell-job &",
			LMCheckpointImagesDir)); err != nil {
		return err
	}
	return nil
}

func (p *LM_Restore) sendCheckpointRequest() (*Response, error) {
	req := &Request{
		Method: "_checkpoint",
		Checkpoint: RequestCheckpoint{
			Name:    p.SrcName,
			DstAddr: p.ThisAddr,
		},
	}
	breq, err := json.Marshal(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	breq = append(breq, []byte("\n")...)
	conn, err := net.Dial("tcp", p.SrcAPIServerAddr)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer conn.Close()
	_, err = conn.Write(breq)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	bresp, err := Readline(conn)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	resp := Response{}
	if err := json.Unmarshal(bresp, &resp); err != nil {
		return nil, errors.WithStack(err)
	}
	return &resp, nil
}

type LM_Checkpoint struct {
	Clientset     kubernetes.Interface
	RestConfig    *rest.Config
	ThisAddr      string
	Namespace     string
	PodName       string
	ContainerName string
	DstAddr       string
}

func (p *LM_Checkpoint) Exec() error {
	sshCloseChan := make(chan struct{})
	closeSSH := func() {
		if sshCloseChan != nil {
			close(sshCloseChan)
			sshCloseChan = nil
		}
	}
	defer closeSSH()
	sshClient, err := NewSSHClient(TheAPICore.HostConf)
	if err != nil {
		return err
	}
	hostEndAddr := fmt.Sprintf("%s:%d", p.ThisAddr, HostLiveMigrationPort)
	remoteEndAddr := fmt.Sprintf("%s:%d", p.DstAddr, HostLiveMigrationPort)
	Logger.Debug("[Checkpoint] open SSH tunnel")
	if err := sshClient.OpenTunnel(hostEndAddr, remoteEndAddr, sshCloseChan); err != nil {
		return err
	}
	Logger.Debug("[Checkpoint] exec mkdir")
	if err := ExecutePod(p.Clientset, p.RestConfig, p.Namespace, p.PodName, p.ContainerName, nil, os.Stdout, os.Stderr,
		"/bin/sh", "-c", fmt.Sprintf("mkdir -p %s", LMCheckpointImagesDir)); err != nil {
		return err
	}
	Logger.Debug("[Checkpoint] get main pid")
	pid, err := p.getMainPid()
	if err != nil {
		return err
	}
	Logger.Debug("[Checkpoint] exec criu dump")
	if err := ExecutePod(p.Clientset, p.RestConfig, p.Namespace, p.PodName, p.ContainerName, nil, os.Stdout, os.Stderr,
		"/bin/sh", "-c", fmt.Sprintf(
			"criu dump --tree %d --images-dir %s --tcp-established --shell-job",
			pid, LMCheckpointImagesDir)); err != nil {
		return err
	}
	Logger.Debug("[Checkpoint] exec rsync")
	if err := ExecutePod(p.Clientset, p.RestConfig, p.Namespace, p.PodName, p.ContainerName, nil, os.Stdout, os.Stderr,
		"/bin/sh", "-c", fmt.Sprintf(
			"rsync -rlOtcv %s/ rsync://%s:%d/%s",
			LMRsyncModuleDirectory, p.ThisAddr, HostLiveMigrationPort, LMRsyncModuleName)); err != nil {
		return err
	}
	return nil
}

func (p *LM_Checkpoint) getMainPid() (int, error) {
	v, err := ReadPodFile(p.Clientset, p.RestConfig, p.Namespace, p.PodName, p.ContainerName, os.Stderr,
		MainPidFilePath)
	if err != nil {
		return 0, err
	}
	ans, err := strconv.Atoi(v)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return ans, nil
}
