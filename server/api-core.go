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
	// TODO: fwdsvc のキーは (name, hostAddr) であるべきだ
	fwdsvcs *sync.Map
}

func NewAPICore(hostConf *HostConf, hostAddr string) *APICore {
	return &APICore{
		HostConf: hostConf,
		HostAddr: hostAddr,
		fwdsvcs:  &sync.Map{},
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
	clientset, _, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	newPod := p.createNewPod(clientset, name, podName, containerName, image, portIn, env, nil, nil)
	clusterIP := p.createOrGetClusterIP(clientset, name, serviceName, clusterIPName, portIn)
	if _, ok := p.fwdsvcs.Load(name); ok {
		Logger.Info("Use existing forwarding service")
	} else if clusterIP != "" {
		clientAddr, appAddr, err := p.getForwardAddrs(portExt, clusterIP, portIn)
		if err == nil {
			if f, err := StartForwardingService("tcp", clientAddr, appAddr, false); err == nil {
				p.fwdsvcs.Store(name, f)
			} else {
				Logger.ErrorE(err)
			}
		} else {
			Logger.ErrorE(err)
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
	if _, ok := p.fwdsvcs.Load(name); ok {
		Logger.Info("Use existing forwarding service")
	} else {
		clientAddr, remoteAddr, err := p.getForwardAddrs(portExt, srcAddr, portIn)
		if err == nil {
			if f, err := StartForwardingService("tcp", clientAddr, remoteAddr, true); err == nil {
				p.fwdsvcs.Store(name, f)
			} else {
				Logger.ErrorE(err)
			}
		} else {
			Logger.ErrorE(err)
		}
	}
}

func (p *APICore) DeployLM(req *Request) {
	namespace := "default"
	name := req.Deploy.Name
	image := req.Deploy.LM.Image
	portIn := int32(req.Deploy.LM.Port.In)
	portExt := int32(req.Deploy.LM.Port.Ext)
	srcAddr := req.Deploy.LM.SrcAddr
	srcName := req.Deploy.LM.SrcName
	podName := ToPodName(name)
	containerName := ToContainerName(name)
	serviceName := ToServiceName(name)
	clusterIPName := ToClusterIPName(name)
	clientset, config, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	command, args := GetRestorePodCommand()
	newPod := p.createNewPod(clientset, name, podName, containerName, image, portIn, nil,
		command, args)
	clusterIP := p.createOrGetClusterIP(clientset, name, serviceName, clusterIPName, portIn)
	if _, ok := p.fwdsvcs.Load(name); ok {
		Logger.Info("Use existing forwarding service")
	} else if clusterIP != "" {
		clientAddr, appAddr, err := p.getForwardAddrs(portExt, clusterIP, portIn)
		if err == nil {
			if f, err := StartForwardingService("tcp", clientAddr, appAddr, false); err == nil {
				p.fwdsvcs.Store(name, f)
			} else {
				Logger.ErrorE(err)
			}
		} else {
			Logger.ErrorE(err)
		}
	} else {
		Logger.Error("Forwarding service cannot not started because ClusterIP is unknown")
	}
	if newPod {
		if err := WaitForPodReady(clientset, podName, WaitPodTimeout); err != nil {
			Logger.ErrorE(err)
		}
		srcAPIServerAddr := fmt.Sprintf("%s:%d", srcAddr, APIServerPort)
		restore := &LM_Restore{
			HostConf:         p.HostConf,
			Clientset:        clientset,
			RestConfig:       config,
			ThisAddr:         p.HostAddr,
			DstNamespace:     namespace,
			DstPodName:       podName,
			DstContainerName: containerName,
			SrcAddr:          srcAddr,
			SrcAPIServerAddr: srcAPIServerAddr,
			SrcName:          srcName,
		}
		if err := restore.Exec(); err != nil {
			Logger.ErrorE(err)
		}
	} else {
		Logger.Warn("Live migration was not performed because creating pod failed")
	}
}

func (p *APICore) Checkpoint(req *Request) *Response {
	namespace := "default"
	name := req.Checkpoint.Name
	srcHostAddr := p.HostAddr
	dstHostAddr := req.Checkpoint.DstAddr
	podName := ToPodName(name)
	containerName := ToContainerName(name)
	clientset, config, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return &Response{Ok: false, Msg: err.Error()}
	}
	checkpoint := &LM_Checkpoint{
		Clientset:     clientset,
		RestConfig:    config,
		ThisAddr:      srcHostAddr,
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,
		DstAddr:       dstHostAddr,
	}
	if err := checkpoint.Exec(); err != nil {
		Logger.ErrorE(err)
		return &Response{Ok: false, Msg: err.Error()}
	}
	return &Response{Ok: true, Msg: ""}
}

func (p *APICore) Remove(req *Request) {
	name := req.Remove.Name
	podName := name + "-pod"
	serviceName := name + "-svc"
	clientset, _, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	if v, ok := p.fwdsvcs.Load(name); ok {
		if err := v.(*ForwardingService).Close(); err != nil {
			Logger.ErrorE(err)
		}
		p.fwdsvcs.Delete(name)
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
