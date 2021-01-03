package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

// TODO: デプロイごとにエントリを持つようにしたい

const (
	WaitPodTimeout = 30 * time.Second
)

type APICore struct {
	HostConf *HostConf
	HostAddr string
	resmap   *sync.Map
}

type DeployResource struct {
	mux    sync.Mutex
	fwdsvc *ForwarderService
}

func NewAPICore(
	hostConf *HostConf,
	hostAddr string,
) *APICore {
	return &APICore{
		HostConf: hostConf,
		HostAddr: hostAddr,
		resmap:   &sync.Map{},
	}
}

func (p *APICore) Deploy(req *Request) {
	switch req.Deploy.Type {
	case DeployTypeNew:
		p.DeployNew(req)
	case DeployTypeFwd:
		p.DeployFwd(req)
	case DeployTypeLM:
		p.DeployLM(req)
	case DeployTypeFwdLM:
		p.DeployFwdLM(req)
	default:
		Logger.Error("Unsupported deploy type: " + req.Deploy.Type)
	}
}

func (p *APICore) DeployNew(req *Request) {
	name := req.Deploy.Name
	image := req.Deploy.NewApp.Image
	portIn := int32(req.Deploy.NewApp.Port.In)
	portExt := int32(req.Deploy.NewApp.Port.Ext)
	env := req.Deploy.NewApp.Env
	podName := ToPodName(name)
	containerName := ToContainerName(name)
	serviceName := ToServiceName(name)
	clusterIPName := ToClusterIPName(name)
	val, _ := p.resmap.LoadOrStore(name, &DeployResource{})
	res := val.(*DeployResource)
	defer res.mux.Unlock()
	res.mux.Lock()
	clientset, _, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	newPod := p.createNewPod(clientset, name, podName, containerName, image, portIn, env, nil, nil)
	clusterIP := p.createOrGetClusterIP(clientset, name, serviceName, clusterIPName, portIn)
	if res.fwdsvc != nil {
		Logger.Info("Use existing forwarding service")
	} else if clusterIP != "" {
		clientAddr, appAddr, err := p.getForwardAddrs(portExt, clusterIP, portIn)
		if err != nil {
			Logger.ErrorE(err)
		} else {
			if fsv, err := StartForwarderService("tcp", clientAddr, appAddr, false); err != nil {
				Logger.ErrorE(err)
			} else {
				res.fwdsvc = fsv
			}
		}
	} else {
		Logger.Error("Forwarding service cannot not started because ClusterIP is unknown")
	}
	if newPod {
		if err := WaitForPodReady(clientset, podName, WaitPodTimeout); err != nil {
			Logger.ErrorE(err)
		}
	}
}

func (p *APICore) DeployFwd(req *Request) {
	name := req.Deploy.Name
	srcAddr := req.Deploy.Fwd.SrcAddr
	portIn := int32(req.Deploy.Fwd.Port.In)
	portExt := int32(req.Deploy.Fwd.Port.Ext)
	val, _ := p.resmap.LoadOrStore(name, &DeployResource{})
	res := val.(*DeployResource)
	defer res.mux.Unlock()
	res.mux.Lock()
	if res.fwdsvc != nil {
		Logger.Info("Use existing forwarding service")
	} else {
		clientAddr, remoteAddr, err := p.getForwardAddrs(portExt, srcAddr, portIn)
		if err != nil {
			Logger.ErrorE(err)
		} else {
			if fsv, err := StartForwarderService("tcp", clientAddr, remoteAddr, true); err != nil {
				Logger.ErrorE(err)
			} else {
				res.fwdsvc = fsv
			}
		}
	}
}

func (p *APICore) DeployLM(req *Request) {
	namespace := "default"
	name := req.Deploy.Name
	image := req.Deploy.LM.Image
	portIn := int32(req.Deploy.LM.Port.In)
	portExt := int32(req.Deploy.LM.Port.Ext)
	env := req.Deploy.LM.Env
	srcAddr := req.Deploy.LM.SrcAddr
	srcName := req.Deploy.LM.SrcName
	interDstAddr := req.Deploy.LM.DstAddr
	bwLimit := req.Deploy.LM.BwLimit
	iteration := req.Deploy.LM.Iteration
	podName := ToPodName(name)
	containerName := ToContainerName(name)
	serviceName := ToServiceName(name)
	clusterIPName := ToClusterIPName(name)
	val, _ := p.resmap.LoadOrStore(name, &DeployResource{})
	res := val.(*DeployResource)
	defer res.mux.Unlock()
	res.mux.Lock()
	clientset, config, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	command, args := GetRestorePodCommand()
	newPod := p.createNewPod(clientset, name, podName, containerName, image, portIn, env,
		command, args)
	clusterIP := p.createOrGetClusterIP(clientset, name, serviceName, clusterIPName, portIn)
	if res.fwdsvc != nil {
		Logger.Info("Use existing forwarding service")
	} else if clusterIP != "" {
		clientAddr, appAddr, err := p.getForwardAddrs(portExt, clusterIP, portIn)
		if err != nil {
			Logger.ErrorE(err)
		} else {
			if fsv, err := StartForwarderService("tcp", clientAddr, appAddr, false); err != nil {
				Logger.ErrorE(err)
			} else {
				res.fwdsvc = fsv
			}
		}
	} else {
		Logger.Error("Forwarding service cannot not started because ClusterIP is unknown")
	}
	if newPod {
		if err := WaitForPodReady(clientset, podName, WaitPodTimeout); err != nil {
			Logger.ErrorE(err)
		} else {
			var thisAddr string
			if interDstAddr != "" {
				thisAddr = interDstAddr
			} else {
				thisAddr = p.HostAddr
			}
			srcAPIServerAddr := fmt.Sprintf("%s:%d", srcAddr, APIServerPort)
			restore := &LM_Restore{
				HostConf:         p.HostConf,
				Clientset:        clientset,
				RestConfig:       config,
				ThisAddr:         thisAddr,
				DstNamespace:     namespace,
				DstPodName:       podName,
				DstContainerName: containerName,
				SrcAddr:          srcAddr,
				SrcAPIServerAddr: srcAPIServerAddr,
				SrcName:          srcName,
				BwLimit:          bwLimit,
				Iteration:        iteration,
			}
			if err := restore.ExecLM(); err != nil {
				Logger.ErrorE(err)
			}
		}
	} else {
		Logger.Warn("Live migration was not performed because creating pod failed")
	}
}

func (p *APICore) DeployFwdLM(req *Request) {
	namespace := "default"
	name := req.Deploy.Name
	image := req.Deploy.FwdLM.Image
	portIn := int32(req.Deploy.FwdLM.Port.In)
	portExt := int32(req.Deploy.FwdLM.Port.Ext)
	env := req.Deploy.FwdLM.Env
	srcAddr := req.Deploy.FwdLM.SrcAddr
	srcPort := int32(req.Deploy.FwdLM.SrcPort)
	srcName := req.Deploy.FwdLM.SrcName
	interDstAddr := req.Deploy.FwdLM.DstAddr
	bwLimit := req.Deploy.FwdLM.BwLimit
	iteration := req.Deploy.FwdLM.Iteration
	dataRate := req.Deploy.FwdLM.DataRate
	podName := ToPodName(name)
	containerName := ToContainerName(name)
	serviceName := ToServiceName(name)
	clusterIPName := ToClusterIPName(name)
	val, _ := p.resmap.LoadOrStore(name, &DeployResource{})
	res := val.(*DeployResource)
	defer res.mux.Unlock()
	res.mux.Lock()
	if res.fwdsvc != nil {
		Logger.Info("Use existing forwarding service")
	} else {
		clientAddr, remoteAddr, err := p.getForwardAddrs(portExt, srcAddr, srcPort)
		if err != nil {
			Logger.ErrorE(err)
		} else {
			if fsv, err := StartForwarderServiceDR("tcp", clientAddr, remoteAddr, true, dataRate); err != nil {
				Logger.ErrorE(err)
			} else {
				res.fwdsvc = fsv
			}
		}
	}
	go func() {
		clientset, config, err := NewClient()
		if err != nil {
			Logger.ErrorE(err)
			return
		}
		command, args := GetRestorePodCommand()
		newPod := p.createNewPod(clientset, name, podName, containerName, image, portIn, env,
			command, args)
		clusterIP := p.createOrGetClusterIP(clientset, name, serviceName, clusterIPName, portIn)
		if !newPod {
			Logger.Warn("Live migration was not performed because creating pod failed")
			return
		}
		if err := WaitForPodReady(clientset, podName, WaitPodTimeout); err != nil {
			Logger.ErrorE(err)
			return
		}
		var thisAddr string
		if interDstAddr != "" {
			thisAddr = interDstAddr
		} else {
			thisAddr = p.HostAddr
		}
		srcAPIServerAddr := fmt.Sprintf("%s:%d", srcAddr, APIServerPort)
		dstPodAddr := fmt.Sprintf("%s:%d", clusterIP, portIn)
		dstPodTCPAddr, err := net.ResolveTCPAddr("tcp", dstPodAddr)
		if err != nil {
			Logger.ErrorE(errors.WithStack(err))
			return
		}
		restore := &LM_Restore{
			HostConf:         p.HostConf,
			Clientset:        clientset,
			RestConfig:       config,
			ThisAddr:         thisAddr,
			DstNamespace:     namespace,
			DstPodName:       podName,
			DstContainerName: containerName,
			SrcAddr:          srcAddr,
			SrcAPIServerAddr: srcAPIServerAddr,
			SrcName:          srcName,
			Fwdsvc:           res.fwdsvc,
			DstPodAddr:       dstPodTCPAddr,
			BwLimit:          bwLimit,
			Iteration:        iteration,
		}
		if err := restore.ExecFwdLM(); err != nil {
			Logger.ErrorE(err)
		}
	}()
}

func (p *APICore) DumpStart(req *Request) *Response {
	namespace := "default"
	name := req.DumpStart.Name
	srcHostAddr := p.HostAddr
	dstHostAddr := req.DumpStart.DstAddr
	bwLimit := req.DumpStart.BwLimit
	podName := ToPodName(name)
	containerName := ToContainerName(name)
	val, _ := p.resmap.LoadOrStore(name, &DeployResource{})
	res := val.(*DeployResource)
	defer res.mux.Unlock()
	res.mux.Lock()
	clientset, config, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return &Response{Ok: false, Msg: err.Error()}
	}
	dump := &LM_DumpService{
		Clientset:     clientset,
		RestConfig:    config,
		ThisAddr:      srcHostAddr,
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,
		DstAddr:       dstHostAddr,
		BwLimit:       bwLimit,
	}
	if err := dump.Start(); err != nil {
		Logger.ErrorE(err)
		return &Response{Ok: false, Msg: err.Error()}
	}
	return &Response{Ok: true, Msg: ""}
}

func (p *APICore) Remove(req *Request) {
	name := req.Remove.Name
	podName := name + "-pod"
	serviceName := name + "-svc"
	val, _ := p.resmap.LoadOrStore(name, &DeployResource{})
	res := val.(*DeployResource)
	defer res.mux.Unlock()
	res.mux.Lock()
	clientset, _, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	if res.fwdsvc != nil {
		if err := res.fwdsvc.Close(); err != nil {
			Logger.Warn(err.Error())
		}
		res.fwdsvc = nil
	}
	if err, errStack := DeleteService(clientset, serviceName); err != nil {
		if k8serrors.IsNotFound(err) {
			Logger.Info("No services to delete")
		} else {
			Logger.ErrorE(errStack)
		}
	} else {
		Logger.Info("Deleting service: " + serviceName)
	}
	delPod := false
	if err, errStack := DeletePod(clientset, podName); err != nil {
		if k8serrors.IsNotFound(err) {
			Logger.Info("No pods to delete")
		} else {
			Logger.ErrorE(errStack)
		}
	} else {
		Logger.Info("Deleting pod: " + podName)
		delPod = true
	}
	if delPod {
		if err := WaitForPodDeleted(clientset, podName, WaitPodTimeout); err != nil {
			Logger.ErrorE(err)
		}
	}
}

func (p *APICore) createNewPod(
	clientset kubernetes.Interface,
	label string,
	podName string,
	containerName string,
	image string,
	containerPort int32,
	env map[string]string,
	command []string,
	args []string,
) bool {
	newPod := false
	pod, err, errStack := CreatePod(clientset, podName, label, containerName, image,
		containerPort, env, command, args)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			Logger.Info("Use existing pod: " + podName)
		} else {
			Logger.ErrorE(errStack)
		}
	} else {
		newPod = true
		Logger.Info("Creating pod: " + pod.GetName())
	}
	return newPod
}

func (p *APICore) createOrGetClusterIP(
	clientset kubernetes.Interface,
	label string,
	serviceName string,
	clusterIPName string,
	containerPort int32,
) string {
	clusterIP := ""
	svc, err, errStack := GetService(clientset, serviceName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			svc, _, errStack := CreateService(clientset, serviceName, label, clusterIPName,
				containerPort)
			if errStack != nil {
				Logger.ErrorE(errStack)
			} else {
				Logger.Info("Creating service: " + svc.GetName())
				clusterIP = svc.Spec.ClusterIP
			}
		} else {
			Logger.ErrorE(errStack)
		}
	} else {
		Logger.Info("Use existing service: " + svc.GetName())
		clusterIP = svc.Spec.ClusterIP
	}
	Logger.DebugF("ClusterIP: %v\n", clusterIP)
	return clusterIP
}

func (p *APICore) getForwardAddrs(
	clientPort int32,
	remoteAddr string,
	remotePort int32,
) (*net.TCPAddr, *net.TCPAddr, error) {
	caddr := fmt.Sprintf(":%d", clientPort)
	ctcpaddr, err := net.ResolveTCPAddr("tcp", caddr)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	raddr := fmt.Sprintf("%s:%d", remoteAddr, remotePort)
	rtcpaddr, err := net.ResolveTCPAddr("tcp", raddr)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return ctcpaddr, rtcpaddr, nil
}

type PortMap struct {
	In  int32
	Ext int32
}

func NewPortMap(in, ext int) *PortMap {
	return &PortMap{
		In:  int32(in),
		Ext: int32(ext),
	}
}

func ToPodName(name string) string {
	podName := name + "-pod"
	return podName
}

func ToContainerName(name string) string {
	containerName := name + "-c"
	return containerName
}

func ToServiceName(name string) string {
	serviceName := name + "-svc"
	return serviceName
}

func ToClusterIPName(name string) string {
	clusterIPName := name + "-cip"
	return clusterIPName
}
