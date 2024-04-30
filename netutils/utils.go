package netutils

import (
	"errors"
	"math/rand"
	"net"
	"time"
)

func GetIPAddress(domain string) (string, bool, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return "", false, err
	}

	for _, ip := range ips {
		if ip.To4() == nil {
			return ip.String(), true, nil
		}
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String(), false, nil
		}
	}

	return "", false, net.InvalidAddrError("No valid IP addresses found")
}

func RandomV6(network string) (net.IP, error) {
	_, subnet, err := net.ParseCIDR(network)
	if err != nil {
		return nil, err
	}

	ones, bits := subnet.Mask.Size()
	if bits != 128 {
		return nil, errors.New("expected an IPv6 network")
	}

	prefix := subnet.IP.To16()

	rand.Seed(time.Now().UnixNano())
	for i := ones; i < bits; i++ {
		byteIndex := i / 8
		bitIndex := uint(i % 8)
		prefix[byteIndex] |= byte(rand.Intn(2)) << (7 - bitIndex)
	}

	return prefix, nil
}
