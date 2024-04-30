package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

func main() {
	flag.IntVar(&port, "port", 33300, "server port")
	flag.StringVar(&cidr, "cidr", "", "ipv6 cidr")
	flag.Parse()

	if cidr == "" {
		log.Fatal("cidr is required")
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	proxy.OnRequest().HijackConnect(
		func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {

			host := req.URL.Hostname()
			_, isV6, err := getIPAddress(host)
			if err != nil {
				log.Printf("Get IP address error: %v", err)
				return
			}

			if !isV6 {
				handleDirectConnection(req, client)
			} else {
				outgoingIP, err := randomV6(cidr)
				if err != nil {
					log.Printf("Generate random IPv6 error: %v", err)
					return
				}

				dialer := &net.Dialer{
					LocalAddr: &net.TCPAddr{IP: net.ParseIP(outgoingIP.String()), Port: 0},
				}

				server, err := dialer.Dial("tcp", req.URL.Host)
				if err != nil {
					errorResponse := fmt.Sprintf("%s 500 Internal Server Error\r\n\r\n", req.Proto)
					client.Write([]byte(errorResponse))
					client.Close()
					return
				}

				okResponse := fmt.Sprintf("%s 200 OK\r\n\r\n", req.Proto)
				client.Write([]byte(okResponse))

				proxyClientServer(client, server)
			}
		},
	)

	proxy.OnRequest().DoFunc(
		func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			host := req.URL.Hostname()
			_, isV6, err := getIPAddress(host)
			if err != nil {
				log.Printf("Get IP address error: %v", err)
				return req, nil
			}

			var localAddr *net.TCPAddr

			if isV6 {
				outgoingIP, err := randomV6(cidr)
				if err != nil {
					log.Printf("Generate random IPv6 error: %v", err)
					return nil, nil
				}

				localAddr = &net.TCPAddr{IP: net.ParseIP(outgoingIP.String()), Port: 0}
			} else {
				localAddr = nil
			}

			dialer := net.Dialer{
				LocalAddr: localAddr,
			}

			newReq, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
			if err != nil {
				log.Printf("New request error: %v", err)
				return req, nil
			}

			newReq.Header = req.Header

			client := &http.Client{
				Transport: &http.Transport{
					DialContext: dialer.DialContext,
				},
			}

			resp, err := client.Do(newReq)
			if err != nil {
				log.Printf("[http] Send request error: %v", err)
				return req, nil
			}
			return req, resp
		},
	)

	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), proxy)

	if err != nil {
		log.Fatal(err)
	}

}

func proxyClientServer(client, server net.Conn) {
	go func() {
		defer server.Close()
		defer client.Close()
		io.Copy(server, client)
	}()
	go func() {
		defer server.Close()
		defer client.Close()
		io.Copy(client, server)
	}()
}

func handleDirectConnection(req *http.Request, client net.Conn) {
	server, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		errorResponse := fmt.Sprintf("%s 500 Internal Server Error\r\n\r\n", req.Proto)
		client.Write([]byte(errorResponse))
		client.Close()
		return
	}
	okResponse := fmt.Sprintf("%s 200 OK\r\n\r\n", req.Proto)
	client.Write([]byte(okResponse))
	proxyClientServer(client, server)
}

func getIPAddress(domain string) (ip string, ipv6 bool, err error) {
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

func randomV6(network string) (net.IP, error) {
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
