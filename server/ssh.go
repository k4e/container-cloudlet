package main

import (
	"io"
	"io/ioutil"
	"net"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	localServerAddr string
	clientConfig    *ssh.ClientConfig
}

func NewSSHClient(hostConf *HostConf) (*SSHClient, error) {
	key, err := ioutil.ReadFile(hostConf.SSHKeyPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	clientConfig := &ssh.ClientConfig{
		User: hostConf.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return &SSHClient{
		localServerAddr: hostConf.SSHLocalServerAddr,
		clientConfig:    clientConfig,
	}, nil
}

func (p *SSHClient) OpenTunnel(
	localAddr string,
	remoteAddr string,
	closeChan chan struct{},
) error {
	sshClientConn, err := ssh.Dial("tcp", p.localServerAddr, p.clientConfig)
	if err != nil {
		return errors.WithStack(err)
	}
	Logger.Info("Open SSH client")
	go func() {
		<-closeChan
		Logger.Info("Close SSH client")
		sshClientConn.Close()
	}()
	ln, err := net.Listen("tcp", localAddr)
	if err != nil {
		return errors.WithStack(err)
	}
	listening := true
	go func() {
		<-closeChan
		listening = false
		ln.Close()
	}()
	go func() {
		for listening {
			localConn, err := ln.Accept()
			if err != nil {
				if IsClosedError(err) {
					return
				} else {
					Logger.ErrorE(err)
					return
				}
			}
			go func() {
				Logger.InfoF("Open SSH tunnel route: %s <--> %s\n", localAddr, remoteAddr)
				remoteConn, err := sshClientConn.Dial("tcp", remoteAddr)
				if err != nil {
					Logger.ErrorE(err)
					return
				}
				go func() {
					<-closeChan
					Logger.InfoF("Close SSH tunnel route: %s <--> %s\n", localAddr, remoteAddr)
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
	}()
	return nil
}
