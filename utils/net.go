package utils

import (
	"fmt"
	"net"
)

func HostToIPv4(host string) (net.IP, error) {
	IPs, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	for _, IP := range IPs {
		if IP.To4() != nil {
			return IP, nil
		}
	}
	return nil, fmt.Errorf("Could not resolve host to IPv4")
}
