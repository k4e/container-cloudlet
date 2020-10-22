package main

import (
	"context"
	"time"

	"github.com/pkg/errors"

	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	PollInterval = 100 * time.Millisecond
)

func NewClient() (kubernetes.Interface, *rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	return clientset, config, nil
}

func CreatePod(
	clientset kubernetes.Interface,
	podName string,
	label string,
	containerName string,
	image string,
	containerPort int32,
	env map[string]string,
	command []string,
	args []string,
) (*apiv1.Pod, error, error) {
	var envVars []apiv1.EnvVar
	for k, v := range env {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	shareProcessNamespace := true
	privileded := true
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
					Command:         command,
					Args:            args,
					Ports: []apiv1.ContainerPort{
						{
							ContainerPort: containerPort,
						},
					},
					Env: envVars,
					SecurityContext: &apiv1.SecurityContext{
						Privileged: &privileded,
					},
				},
			},
			ShareProcessNamespace: &shareProcessNamespace,
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{
					Name: "regcred",
				},
			},
			RestartPolicy: apiv1.RestartPolicyNever,
		},
	}
	result, err := clientset.CoreV1().Pods("default").
		Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err, errors.WithStack(err)
	}
	return result, nil, nil
}

func GetPod(
	clientset kubernetes.Interface,
	podName string,
) (*apiv1.Pod, error, error) {
	pod, err := clientset.CoreV1().Pods("default").Get(
		context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, err, errors.WithStack(err)
	}
	return pod, nil, nil
}

func DeletePod(
	clientset kubernetes.Interface,
	podName string,
) (error, error) {
	namespace := "default"
	err := clientset.CoreV1().Pods(namespace).Delete(
		context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		return err, errors.WithStack(err)
	}
	return nil, nil
}

func IsPodReady(clientset kubernetes.Interface, podName string) (bool, error) {
	pod, err, _ := GetPod(clientset, podName)
	if err != nil {
		return false, err
	}
	ans := (pod.Status.Phase != apiv1.PodPending)
	return ans, nil
}

func WaitForPodReady(
	clientset kubernetes.Interface,
	podName string,
	timeout time.Duration,
) error {
	condFunc := func() (bool, error) {
		return IsPodReady(clientset, podName)
	}
	return wait.PollImmediate(PollInterval, timeout, condFunc)
}

func IsPodDeleted(clientset kubernetes.Interface, podName string) (bool, error) {
	_, err, _ := GetPod(clientset, podName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return true, nil
		} else {
			return false, err
		}
	}
	return false, nil
}

func WaitForPodDeleted(
	clientset kubernetes.Interface,
	podName string,
	timeout time.Duration,
) error {
	condFunc := func() (bool, error) {
		return IsPodDeleted(clientset, podName)
	}
	return wait.PollImmediate(PollInterval, timeout, condFunc)
}

func CreateService(
	clientset kubernetes.Interface,
	serviceName string,
	label string,
	clusterIpName string,
	port int32,
) (*apiv1.Service, error, error) {
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
		return nil, err, errors.WithStack(err)
	}
	return result, nil, nil
}

func GetService(
	clientset kubernetes.Interface,
	serviceName string,
) (*apiv1.Service, error, error) {
	svc, err := clientset.CoreV1().Services("default").
		Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, err, errors.WithStack(err)
	}
	return svc, nil, nil
}

func DeleteService(
	clientset kubernetes.Interface,
	serviceName string,
) (error, error) {
	err := clientset.CoreV1().Services("default").Delete(
		context.TODO(), serviceName, metav1.DeleteOptions{})
	if err != nil {
		return err, errors.WithStack(err)
	}
	return nil, nil
}
