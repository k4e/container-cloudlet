package main

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
)

func GetInterfaceAddr(networkInterface string) (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", errors.WithStack(err)
	}
	for _, iface := range ifaces {
		if iface.Name == networkInterface {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", errors.WithStack(err)
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
					return ip.To4().String(), nil
				}
			}
		}
	}
	return "", errors.New(fmt.Sprintf("No IPv4 addr for iface: " + networkInterface))
}

func PrintInterfaceAddrs(prefix string) error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return errors.WithStack(err)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return errors.WithStack(err)
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() {
				fmt.Printf("%s%v\t= %v\n", prefix, iface.Name, ip.String())
			}
		}
	}
	return nil
}
