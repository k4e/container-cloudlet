package main

import (
	"io"
	"io/ioutil"
	"net"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

type SSHConf struct {
	SSHLocalServerAddr string `yaml:"sshLocalServerAddr"`
	SSHUser            string `yaml:"sshUser"`
	SSHKeyPath         string `yaml:"sshKeyPath"`
}

func LoadSSHConf(path string) (*SSHConf, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		Logger.Error("SSHClient: fail to read SSH config file")
		return nil, err
	}
	sshConf := &SSHConf{}
	err = yaml.Unmarshal(b, sshConf)
	if err != nil {
		Logger.Error("SSHClient: fail to unmarshal yaml file")
		return nil, err
	}
	return sshConf, nil
}

type SSHClient struct {
	localServerAddr string
	clientConfig    *ssh.ClientConfig
}

func NewSSHClient(sshConf *SSHConf) (*SSHClient, error) {
	key, err := ioutil.ReadFile(sshConf.SSHKeyPath)
	if err != nil {
		Logger.Error("SSHClient: fail to read SSH private key")
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		Logger.Error("SSHClient: fail to parse SSH private key")
		return nil, err
	}
	clientConfig := &ssh.ClientConfig{
		User: sshConf.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return &SSHClient{
		localServerAddr: sshConf.SSHLocalServerAddr,
		clientConfig:    clientConfig,
	}, nil
}

func (p *SSHClient) OpenTunnel(
	localAddr string,
	remoteAddr string,
	chanReady chan struct{},
	chanClose chan struct{},
) error {
	sshClientConn, err := ssh.Dial("tcp", p.localServerAddr, p.clientConfig)
	if err != nil {
		Logger.ErrorE(err)
		return err
	}
	defer func() {
		Logger.Info("Close SSH client")
		sshClientConn.Close()
	}()
	ln, err := net.Listen("tcp", localAddr)
	if err != nil {
		Logger.ErrorE(err)
		return err
	}
	listening := true
	go func() {
		<-chanClose
		listening = false
		ln.Close()
	}()
	chanReady <- struct{}{}
	for listening {
		localConn, err := ln.Accept()
		if err != nil {
			if IsClosedError(err) {
				break
			} else {
				Logger.ErrorE(err)
				return err
			}
		}
		go func() {
			Logger.InfoF("Open SSH tunnel path: %s <--> %s\n", localAddr, remoteAddr)
			remoteConn, err := sshClientConn.Dial("tcp", remoteAddr)
			if err != nil {
				Logger.ErrorE(err)
			}
			go func() {
				<-chanClose
				Logger.InfoF("Close SSH tunnel path: %s <--> %s\n", localAddr, remoteAddr)
				localConn.Close()
				remoteConn.Close()
			}()
			go func() {
				if _, err := io.Copy(remoteConn, localConn); err != nil {
					Logger.ErrorE(err)
				}
			}()
			go func() {
				if _, err := io.Copy(localConn, remoteConn); err != nil {
					Logger.ErrorE(err)
				}
			}()
		}()
	}
	return nil
}
