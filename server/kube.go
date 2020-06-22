package main

import (
	"context"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClient() (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func CreatePod(
	clientset kubernetes.Interface,
	podName string,
	label string,
	containerName string,
	image string,
	containerPort int32,
	env map[string]string,
) (*apiv1.Pod, error) {
	var envVars []apiv1.EnvVar
	for k, v := range env {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	pod := &apiv1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   podName,
			Labels: map[string]string{"app": label},
		},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:            containerName,
					Image:           image,
					ImagePullPolicy: "Always",
					Ports: []apiv1.ContainerPort{
						{
							ContainerPort: containerPort,
						},
					},
					Env: envVars,
				},
			},
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{
					Name: "regcred",
				},
			},
		},
	}
	result, err := clientset.CoreV1().Pods("default").
		Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func CreateService(
	clientset kubernetes.Interface,
	serviceName string,
	label string,
	clusterIpName string,
	port int32,
) (*apiv1.Service, error) {
	service := &apiv1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceName,
			Labels: map[string]string{"app": label},
		},
		Spec: apiv1.ServiceSpec{
			Type: "ClusterIP",
			Ports: []apiv1.ServicePort{
				{
					Name:     clusterIpName,
					Protocol: "TCP",
					Port:     port,
				},
			},
			Selector: map[string]string{"app": label},
		},
	}
	result, err := clientset.CoreV1().Services("default").
		Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetService(
	clientset kubernetes.Interface,
	serviceName string,
) (*apiv1.Service, error) {
	return clientset.CoreV1().Services("default").
		Get(context.TODO(), serviceName, metav1.GetOptions{})
}

func DeletePod(
	clientset kubernetes.Interface,
	podName string,
) error {
	return clientset.CoreV1().Pods("default").Delete(
		context.TODO(), podName, metav1.DeleteOptions{})
}

func DeleteService(
	clientset kubernetes.Interface,
	serviceName string,
) error {
	return clientset.CoreV1().Services("default").Delete(
		context.TODO(), serviceName, metav1.DeleteOptions{})
}
