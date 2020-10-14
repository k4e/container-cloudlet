package main

import (
	"k8s.io/client-go/kubernetes"
)

const CrImagesDir = "/tmp/cloudlet.checkpoint/"
const CrPort = 31213

func ExecLiveMigration(podName, srcAddr string, clientset kubernetes.Interface) error {
	return nil
}

func ExecCheckpoint(podName, dstAddr string, clientset kubernetes.Interface) error {
	return nil
}
