package main

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	HostConfPath = "./hostconf.yaml"
)

type HostConf struct {
	HostNetworkInterface string `yaml:"hostNetworkInterface"`
	SSHLocalServerAddr   string `yaml:"sshLocalServerAddr"`
	SSHUser              string `yaml:"sshUser"`
	SSHKeyPath           string `yaml:"sshKeyPath"`
}

func LoadHostConf() (*HostConf, error) {
	return LoadHostConfFrom(HostConfPath)
}

func LoadHostConfFrom(path string) (*HostConf, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	hostConf := &HostConf{}
	err = yaml.Unmarshal(b, hostConf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return hostConf, nil
}
