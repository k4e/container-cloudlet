package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	LM_HostMsgPort          = 19998
	LM_HostDataPort         = 19999
	LM_PodRsyncPort         = 873
	LM_DumpImagesDir        = "/tmp/cloudlet-live-migration/images"
	LM_RsyncModuleName      = "tmp"
	LM_RsyncModuleDirectory = "/tmp"
	MainPidFilePath         = "/MAIN_PID"
)

const (
	LM_MsgReqPreDump = 0x01
	LM_MsgReqDump    = 0x02
	LM_MsgRespOk     = 0x00
	LM_MsgRespError  = 0xFF
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
	var startTime time.Time
	var endTime time.Time
	Logger.Debug("[Restore] Exec rsync --daemon")
	if err := ExecutePod(p.Clientset, p.RestConfig, p.DstNamespace, p.DstPodName, p.DstContainerName,
		nil, os.Stdout, os.Stderr, "/bin/sh", "-c", "rsync --daemon"); err != nil {
		return err
	}
	Logger.Debug("[Restore] Send DumpStart request")
	if resp, err := p.sendDumpStartRequest(); err != nil {
		return err
	} else if !resp.Ok {
		return errors.New("DumpStart response error: " + resp.Msg)
	}
	dumpServiceAddr := fmt.Sprintf("%s:%d", p.SrcAddr, LM_HostMsgPort)
	conn, err := net.Dial("tcp", dumpServiceAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	k8sPortFwdCloseChan := make(chan struct{})
	defer close(k8sPortFwdCloseChan)
	Logger.Debug("[Restore] Open kube port-forward")
	if err := OpenKubePortForwardReady(p.RestConfig, p.DstNamespace, p.DstPodName,
		LM_HostDataPort, LM_PodRsyncPort, os.Stdout, os.Stderr, k8sPortFwdCloseChan); err != nil {
		return err
	}
	Logger.Debug("[Restore] Send pre-dump request")
	startTime = time.Now()
	if err := p.sendDumpServiceRequest(conn, LM_MsgReqPreDump); err != nil {
		return err
	}
	endTime = time.Now()
	Logger.DebugF("[Restore] Pre-dump time (ns): %d\n", endTime.Sub(startTime).Nanoseconds())
	Logger.Debug("[Restore] Send final dump request")
	startTime = time.Now()
	if err := p.sendDumpServiceRequest(conn, LM_MsgReqDump); err != nil {
		return err
	}
	endTime = time.Now()
	Logger.DebugF("[Restore] Final dump time (ns): %d\n", endTime.Sub(startTime).Nanoseconds())
	Logger.Debug("[Restore] Exec unshare criu restore")
	startTime = time.Now()
	if err := ExecutePod(p.Clientset, p.RestConfig, p.DstNamespace, p.DstPodName, p.DstContainerName,
		nil, os.Stdout, os.Stderr, "/bin/sh", "-c", fmt.Sprintf(
			"unshare -p -m --fork --mount-proc criu restore --images-dir %s/final --tcp-established --shell-job -vvvv &",
			LM_DumpImagesDir)); err != nil {
		return err
	}
	endTime = time.Now()
	Logger.DebugF("[Restore] Restore time (ns): %d\n", endTime.Sub(startTime).Nanoseconds())
	return nil
}

func (p *LM_Restore) sendDumpStartRequest() (*Response, error) {
	req := &Request{
		Method: "_dumpStart",
		DumpStart: RequestDumpStart{
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

func (p *LM_Restore) sendDumpServiceRequest(conn net.Conn, req byte) error {
	reqbuf := []byte{req}
	respbuf := make([]byte, 1)
	var err error
	for {
		if n, err := conn.Write(reqbuf); n > 0 || err != nil {
			break
		}
	}
	if err != nil {
		return errors.WithStack(err)
	}
	for {
		if n, err := conn.Read(respbuf); n > 0 || err != nil {
			break
		}
	}
	if err != nil {
		return errors.WithStack(err)
	} else if respbuf[0] != LM_MsgRespOk {
		return errors.New("Dump service respond an error")
	}
	return nil
}

type LM_DumpService struct {
	Clientset     kubernetes.Interface
	RestConfig    *rest.Config
	ThisAddr      string
	Namespace     string
	PodName       string
	ContainerName string
	DstAddr       string
}

func (p *LM_DumpService) Start() (reterr error) {
	lnAddr := fmt.Sprintf(":%d", LM_HostMsgPort)
	lnTCPAddr, err := net.ResolveTCPAddr("tcp", lnAddr)
	if err != nil {
		return err
	}
	Logger.Debug("[Dump] Open message listener")
	ln, err := net.ListenTCP("tcp", lnTCPAddr)
	if err != nil {
		return err
	}
	defer func() {
		if reterr != nil {
			ln.Close()
		}
	}()
	sshClient, err := NewSSHClient(TheAPICore.HostConf)
	if err != nil {
		return err
	}
	hostEndAddr := fmt.Sprintf("%s:%d", p.ThisAddr, LM_HostDataPort)
	remoteEndAddr := fmt.Sprintf("%s:%d", p.DstAddr, LM_HostDataPort)
	sshCloseChan := make(chan struct{})
	defer func() {
		if reterr != nil {
			close(sshCloseChan)
		}
	}()
	Logger.Debug("[Dump] Open SSH tunnel")
	if err := sshClient.OpenTunnel(hostEndAddr, remoteEndAddr, sshCloseChan); err != nil {
		return err
	}
	Logger.Debug("[Dump] Get main pid")
	pid, err := p.getMainPid()
	if err != nil {
		return err
	}
	go func() {
		defer func() {
			close(sshCloseChan)
			ln.Close()
		}()
		conn, err := ln.Accept()
		if err != nil {
			Logger.ErrorE(errors.WithStack(err))
			return
		}
		defer conn.Close()
		reqbuf := make([]byte, 1)
		for itercnt := 1; true; itercnt++ {
			var n int
			var err error
			for {
				if n, err = conn.Read(reqbuf); n > 0 || err != nil {
					break
				}
			}
			if err != nil {
				if err == io.EOF {
					Logger.Debug("[Dump][svc] Received EOF")
				} else {
					Logger.ErrorE(errors.WithStack(err))
				}
				return
			}
			req := reqbuf[0]
			var resp byte
			if req == LM_MsgReqPreDump {
				argb := strings.Builder{}
				fmt.Fprintf(&argb, "mkdir -p %s/%d", LM_DumpImagesDir, itercnt)
				imagesDir := fmt.Sprintf("%s/%d", LM_DumpImagesDir, itercnt)
				prevImagesDirOpt := ""
				if itercnt > 1 {
					prevImagesDirOpt = fmt.Sprintf("--prev-images-dir ../%d", itercnt-1)
				}
				fmt.Fprintf(&argb, " && criu pre-dump --tree %d --images-dir %s %s --tcp-established --shell-job",
					pid, imagesDir, prevImagesDirOpt)
				fmt.Fprintf(&argb, " && rsync -rlOt %s/ rsync://%s:%d/%s",
					LM_RsyncModuleDirectory, p.ThisAddr, LM_HostDataPort, LM_RsyncModuleName)
				Logger.Debug("[Dump][svc] Exec mkdir && criu pre-dump && rsync")
				if err := ExecutePod(p.Clientset, p.RestConfig, p.Namespace, p.PodName, p.ContainerName,
					nil, os.Stdout, os.Stderr, "/bin/sh", "-c", argb.String()); err != nil {
					Logger.ErrorE(errors.WithStack(err))
					resp = LM_MsgRespError
				} else {
					resp = LM_MsgRespOk
				}
			} else if req == LM_MsgReqDump {
				argb := strings.Builder{}
				fmt.Fprintf(&argb, "mkdir -p %s/final", LM_DumpImagesDir)
				imagesDir := fmt.Sprintf("%s/final", LM_DumpImagesDir)
				prevImagesDirOpt := ""
				if itercnt > 1 {
					prevImagesDirOpt = fmt.Sprintf("--prev-images-dir ../%d", itercnt-1)
				}
				fmt.Fprintf(&argb, " && criu dump --tree %d --images-dir %s %s --tcp-established --shell-job --track-mem",
					pid, imagesDir, prevImagesDirOpt)
				fmt.Fprintf(&argb, " && rsync -rlOt %s/ rsync://%s:%d/%s",
					LM_RsyncModuleDirectory, p.ThisAddr, LM_HostDataPort, LM_RsyncModuleName)
				Logger.Debug("[Dump][svc] Exec mkdir && criu dump && rsync")
				if err := ExecutePod(p.Clientset, p.RestConfig, p.Namespace, p.PodName, p.ContainerName,
					nil, os.Stdout, os.Stderr, "/bin/sh", "-c", argb.String()); err != nil {
					Logger.ErrorE(errors.WithStack(err))
					resp = LM_MsgRespError
				} else {
					resp = LM_MsgRespOk
				}
			} else {
				Logger.ErrorF("[Dump][svc] Unexpected message: %x\n", req)
				resp = LM_MsgRespError
			}
			respbuf := []byte{resp}
			for {
				if n, err = conn.Write(respbuf); n > 0 || err != nil {
					break
				}
			}
			if err != nil {
				Logger.ErrorE(errors.WithStack(err))
				return
			}
		}
	}()
	return nil
}

func (p *LM_DumpService) getMainPid() (int, error) {
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
