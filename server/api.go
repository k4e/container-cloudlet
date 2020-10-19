package main

import (
	"fmt"
	"net"
	"sync"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

type APIService struct {
	// fwdsvc のキーは (name, hostAddr) であるべきだ
	fwdsvcs sync.Map
}

func NewAPIService() *APIService {
	return &APIService{}
}

func (p *APIService) DeployNew(name, image string, port *PortMap, env map[string]string) {
	podName := ToPodName(name)
	containerName := ToContainerName(name)
	serviceName := ToServiceName(name)
	clusterIPName := ToClusterIPName(name)
	clientset, _, err := NewClient()
	if err != nil {
		Logger.ErrorE(err)
		return
	}
	pod, err := CreatePod(clientset, podName, name, containerName, image, port.In, env)
	if err != nil {
		Logger.ErrorE(err)
	} else {
		Logger.Info("Creating pod: " + pod.GetName())
	}
	clusterIP := ""
	svc, err := GetService(clientset, serviceName)
	if err == nil {
		Logger.Info("Use existing service: " + svc.GetName())
		clusterIP = svc.Spec.ClusterIP
	} else {
		if k8serrors.IsNotFound(err) {
			svc, err := CreateService(clientset, serviceName, name, clusterIPName, port.In)
			if err != nil {
				Logger.ErrorE(err)
			} else {
				Logger.Info("Creating service: " + svc.GetName())
				clusterIP = svc.Spec.ClusterIP
			}
		} else {
			Logger.ErrorE(err)
		}
	}
	if _, ok := p.fwdsvcs.Load(name); ok {
		Logger.Info("Use existing forwarding service")
	} else if clusterIP != "" {
		clientAddrStr := fmt.Sprintf(":%d", port.Ext)
		clientAddr, err := net.ResolveTCPAddr("tcp", clientAddrStr)
		if err != nil {
			Logger.ErrorE(err)
		}
		appAddrStr := fmt.Sprintf("%s:%d", clusterIP, port.In)
		appAddr, err := net.ResolveTCPAddr("tcp", appAddrStr)
		if err != nil {
			Logger.ErrorE(err)
		}
		if f, err := StartForwardingService("tcp", clientAddr, appAddr, false); err == nil {
			p.fwdsvcs.Store(name, f)
		} else {
			Logger.ErrorE(err)
		}
	} else {
		Logger.Error("Error: forwarding service cannot not started because ClusterIP is unknown")
	}
}

func (p *APIService) DeployFwd(name, srcAddr string, port *PortMap) {
	if _, ok := p.fwdsvcs.Load(name); ok {
		Logger.Info("Use existing forwarding service")
	} else {
		clientAddrStr := fmt.Sprintf(":%d", port.Ext)
		clientAddr, err := net.ResolveTCPAddr("tcp", clientAddrStr)
		if err != nil {
			Logger.ErrorE(err)
		}
		hostAddrStr := fmt.Sprintf("%s:%d", srcAddr, port.In)
		hostAddr, err := net.ResolveTCPAddr("tcp", hostAddrStr)
		if err != nil {
			Logger.ErrorE(err)
		}
		if f, err := StartForwardingService("tcp", clientAddr, hostAddr, true); err == nil {
			p.fwdsvcs.Store(name, f)
		} else {
			Logger.ErrorE(err)
		}
	}
}

func (p *APIService) Remove(name string) {
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
	if err := DeleteService(clientset, serviceName); err != nil {
		Logger.ErrorE(err)
	} else {
		Logger.Info("Deleting service: " + serviceName)
	}
	if err := DeletePod(clientset, podName); err != nil {
		Logger.ErrorE(err)
	} else {
		Logger.Info("Deleting pod: " + podName)
	}
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
