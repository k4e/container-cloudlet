package main

import (
	"encoding/json"
	stderrors "errors"
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
	LM_HostMsgPort          = 19999
	LM_HostDataPort         = 19998
	LM_HostResumeSigPort    = 19997
	LM_PodRsyncPort         = 873
	LM_DumpImagesDir        = "/tmp/cloudlet-live-migration/images"
	LM_RsyncModuleName      = "tmp"
	LM_RsyncModuleDirectory = "/tmp"
	LM_PostResumeScriptPath = "/tmp/cloudlet-live-migration.post-resume.sh"
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
	Fwdsvc           *ForwarderService
	DstPodAddr       *net.TCPAddr
	BwLimit          int
	Iteration        int
}

func (p *LM_Restore) ExecLM() error {
	return p.exec(false)
}

func (p *LM_Restore) ExecFwdLM() error {
	return p.exec(true)
}

func (p *LM_Restore) exec(withFwd bool) (reterr error) {
	defer func() {
		if reterr != nil {
			Logger.Warn("[Restore] Abort")
		} else {
			Logger.Info("[Restore] Complete")
		}
	}()
	Logger.Info("[Restore] Listen to resume signal")
	lnResumeAddr := fmt.Sprintf(":%d", LM_HostResumeSigPort)
	lnResume, err := net.Listen("tcp", lnResumeAddr)
	if err != nil {
		return errors.WithStack(err)
	}
	defer lnResume.Close()
	resumeChan := make(chan struct{}, 1)
	go func() {
		defer close(resumeChan)
		conn, err := lnResume.Accept()
		if err != nil {
			if IsClosedError(err) {
				Logger.Warn("[Restore] Resume signal listener close")
			} else {
				Logger.ErrorE(err)
			}
			return
		}
		Logger.Info("[Restore] Resume signal accept")
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(1 * time.Second))
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
	}()
	Logger.Info("[Restore] Prepare post-resume script")
	postResumeScript := fmt.Sprintf("#!/bin/sh\n"+
		"if test \"$CRTOOLS_SCRIPT_ACTION\" = \"post-resume\"; then echo Send resume signal; nc -vz %s %d; fi\n",
		p.ThisAddr, LM_HostResumeSigPort)
	if err := WritePodFile(p.Clientset, p.RestConfig, p.DstNamespace, p.DstPodName, p.DstContainerName,
		os.Stderr, LM_PostResumeScriptPath, postResumeScript, "755"); err != nil {
		return err
	}
	Logger.Info("[Restore] Exec rsync --daemon")
	if err := ExecutePod(p.Clientset, p.RestConfig, p.DstNamespace, p.DstPodName, p.DstContainerName,
		nil, os.Stdout, os.Stderr, "/bin/sh", "-c", "rsync --daemon"); err != nil {
		return err
	}
	Logger.Info("[Restore] Send DumpStart request")
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
	Logger.Info("[Restore] Open kube port-forward")
	if err := OpenKubePortForwardReady(p.RestConfig, p.DstNamespace, p.DstPodName,
		LM_HostDataPort, LM_PodRsyncPort, os.Stdout, os.Stderr, k8sPortFwdCloseChan); err != nil {
		return err
	}
	iteration := 1
	if p.Iteration > 0 {
		iteration = p.Iteration
	} else if p.Iteration == 0 {
		iteration = 1
	} else {
		iteration = 0
	}
	Logger.DebugF("[Restore] Pre-dump iteration: %d\n", iteration)
	startPreDump := time.Now()
	for itr := 0; itr < iteration; itr++ {
		Logger.InfoF("[Restore] Send pre-dump request (%d)\n", (itr + 1))
		if err := p.sendDumpServiceRequest(conn, LM_MsgReqPreDump); err != nil {
			return err
		}
	}
	Logger.DebugF("[Restore] Pre-dump time (ms): %d\n", time.Now().Sub(startPreDump).Milliseconds())
	if withFwd {
		Logger.Info("[Restore] Suspend forwarding service")
		p.Fwdsvc.Suspend()
		defer func() {
			Logger.Info("[Restore] Resume forwarding service")
			p.Fwdsvc.Resume()
		}()
		Logger.Info("[Restore] Close all forwarding streams")
		p.Fwdsvc.CloseAllForwarders()
	}
	Logger.Info("[Restore] Send final dump request")
	startFinalDump := time.Now()
	startDowntime := time.Now()
	if err := p.sendDumpServiceRequest(conn, LM_MsgReqDump); err != nil {
		return err
	}
	Logger.DebugF("[Restore] Final dump time (ms): %d\n", time.Now().Sub(startFinalDump).Milliseconds())
	Logger.Info("[Restore] Exec unshare criu restore")
	timeoutChan := make(chan struct{}, 1)
	go func() {
		defer close(timeoutChan)
		time.Sleep(5 * time.Second)
	}()
	go func() {
		actionScriptOpt := fmt.Sprintf("--action-script %s", LM_PostResumeScriptPath)
		if err := ExecutePod(p.Clientset, p.RestConfig, p.DstNamespace, p.DstPodName, p.DstContainerName,
			nil, os.Stdout, os.Stderr, "/bin/sh", "-c", fmt.Sprintf(
				"unshare -p -m --fork --mount-proc"+
					" criu restore --images-dir %s/final --shell-job --tcp-close %s &",
				LM_DumpImagesDir, actionScriptOpt)); err != nil {
			Logger.ErrorE(err)
		}
	}()
	for waitForResume := true; waitForResume; {
		select {
		case <-resumeChan:
			Logger.DebugF("[Restore] Estimated downtime (ms): %d\n", time.Now().Sub(startDowntime).Milliseconds())
			waitForResume = false
		case <-timeoutChan:
			Logger.Warn("[Restore] Waiting for resume timeout")
			waitForResume = false
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	if withFwd {
		Logger.Info("[Restore] Change forwarding dst addr to the restored pod")
		p.Fwdsvc.ChangeServerAddr(p.DstPodAddr)
		p.Fwdsvc.ChangeDataRate(0)
	}
	return nil
}

func (p *LM_Restore) sendDumpStartRequest() (*Response, error) {
	req := &Request{
		Method: "_dumpStart",
		DumpStart: RequestDumpStart{
			Name:    p.SrcName,
			DstAddr: p.ThisAddr,
			BwLimit: p.BwLimit,
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
	if _, err := conn.Write(reqbuf); err != nil {
		return errors.WithStack(err)
	}
	respbuf := make([]byte, 1)
	if n, err := conn.Read(respbuf); !(n > 0) {
		if err != nil {
			return errors.WithStack(err)
		} else {
			return errors.New("Read 0 bytes")
		}
	} else if respbuf[0] != LM_MsgRespOk {
		return stderrors.New("Dump service respond an error")
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
	BwLimit       int
}

func (p *LM_DumpService) Start() (reterr error) {
	defer func() {
		if reterr != nil {
			Logger.Warn("[Dump] Abort")
		} else {
			Logger.Info("[Dump] Complete")
		}
	}()
	lnAddr := fmt.Sprintf(":%d", LM_HostMsgPort)
	lnTCPAddr, err := net.ResolveTCPAddr("tcp", lnAddr)
	if err != nil {
		return err
	}
	Logger.Info("[Dump] Open message listener")
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
	Logger.Info("[Dump] Open SSH tunnel")
	if err := sshClient.OpenTunnel(hostEndAddr, remoteEndAddr, sshCloseChan); err != nil {
		return err
	}
	Logger.Info("[Dump] Get main pid")
	pid, err := p.getMainPid()
	if err != nil {
		return err
	}
	rsyncBw := p.getRsyncBandwidth()
	rsyncBwOpt := ""
	if rsyncBw > 0 {
		rsyncBwOpt = fmt.Sprintf("--bwlimit=%d", rsyncBw)
	}
	go func() {
		defer func() {
			close(sshCloseChan)
			ln.Close()
		}()
		conn, err := ln.Accept()
		if err != nil {
			Logger.ErrorE(err)
			return
		}
		defer conn.Close()
		reqbuf := make([]byte, 1)
		for itercnt := 1; true; itercnt++ {
			if n, err := conn.Read(reqbuf); !(n > 0) {
				if err == io.EOF {
					Logger.Info("[Dump][svc] Received EOF")
				} else if err != nil {
					Logger.ErrorE(err)
				} else {
					Logger.ErrorE(errors.New("Read 0 bytes"))
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
				fmt.Fprintf(&argb, " && criu pre-dump --tree %d --images-dir %s %s --tcp-close --shell-job",
					pid, imagesDir, prevImagesDirOpt)
				fmt.Fprintf(&argb, " && rsync %s -rlOt %s/ rsync://%s:%d/%s",
					rsyncBwOpt, LM_RsyncModuleDirectory, p.ThisAddr, LM_HostDataPort, LM_RsyncModuleName)
				Logger.Info("[Dump][svc] Exec mkdir && criu pre-dump && rsync")
				if rsyncBwOpt != "" {
					Logger.InfoF("[Dump][svc] Rsync bandwidth: %d KiB/s\n", rsyncBw)
				}
				if err := ExecutePod(p.Clientset, p.RestConfig, p.Namespace, p.PodName, p.ContainerName,
					nil, os.Stdout, os.Stderr, "/bin/sh", "-c", argb.String()); err != nil {
					Logger.ErrorE(err)
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
				fmt.Fprintf(&argb, " && criu dump --tree %d --images-dir %s %s --tcp-close --shell-job --track-mem",
					pid, imagesDir, prevImagesDirOpt)
				fmt.Fprintf(&argb, " && rsync -rlOt %s/ rsync://%s:%d/%s",
					LM_RsyncModuleDirectory, p.ThisAddr, LM_HostDataPort, LM_RsyncModuleName)
				Logger.Info("[Dump][svc] Exec mkdir && criu dump && rsync")
				if err := ExecutePod(p.Clientset, p.RestConfig, p.Namespace, p.PodName, p.ContainerName,
					nil, os.Stdout, os.Stderr, "/bin/sh", "-c", argb.String()); err != nil {
					Logger.ErrorE(err)
					resp = LM_MsgRespError
				} else {
					resp = LM_MsgRespOk
				}
			} else {
				Logger.ErrorF("[Dump][svc] Unexpected message: %x\n", req)
				resp = LM_MsgRespError
			}
			respbuf := []byte{resp}
			if _, err := conn.Write(respbuf); err != nil {
				Logger.ErrorE(err)
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

func (p *LM_DumpService) getRsyncBandwidth() int {
	if p.BwLimit <= 0 {
		return 0
	}
	return p.BwLimit * 122
}
