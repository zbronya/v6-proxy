package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os/exec"
	"os/user"
	"strings"
	"time"
)

var port int

var cidr string

var bind string

var autoRoute bool

var autoForwarding bool

type AuthConfig struct {
	username string
	password string
}

func main() {

	authConfig := AuthConfig{}

	flag.IntVar(&port, "port", 33300, "server port")
	flag.StringVar(&cidr, "cidr", "", "ipv6 cidr")
	flag.StringVar(&authConfig.username, "username", "", "Basic auth username")
	flag.StringVar(&authConfig.password, "password", "", "Basic auth password")
	flag.StringVar(&bind, "bind", "127.0.0.1", "Bind address")
	flag.BoolVar(&autoRoute, "auto-route", true, "Auto add route to local network")
	flag.BoolVar(&autoForwarding, "auto-forwarding", true, "Auto enable ipv6 forwarding")
	flag.Parse()

	if cidr == "" {
		log.Fatal("cidr is required")
	}

	if autoForwarding {
		if isRoot() {
			setV6Forwarding()
		} else {
			log.Fatal("You must run this program as root")
		}
	}

	if autoRoute {
		if isRoot() {
			addV6Route(cidr)
		} else {
			log.Fatal("You must run this program as root")
		}
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	proxy.OnRequest().DoFunc(
		func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			if authConfig.username != "" && authConfig.password != "" && !authConfig.checkAuth(req) {
				return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusProxyAuthRequired, "Proxy Authentication Required")
			}
			return req, nil
		},
	)

	proxy.OnRequest().HijackConnect(
		func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
			if authConfig.username != "" && authConfig.password != "" && !authConfig.checkAuth(req) {
				client.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"Proxy\"\r\n\r\n"))
				client.Close()
				return
			}

			host := req.URL.Hostname()
			targetIp, isV6, err := getIPAddress(host)
			if err != nil {
				log.Printf("Get IP address error: %v", err)
				return
			}

			if !isV6 {
				log.Printf("Connecting to %s [%s] from local net", req.URL.Host, targetIp)
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

				log.Printf("Connecting to %s [%s] from %s", req.URL.Host, targetIp, outgoingIP.String())

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
			targetIp, isV6, err := getIPAddress(host)
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

				log.Printf("Connecting to %s [%s] from %s", req.URL.Host, targetIp, outgoingIP.String())
				localAddr = &net.TCPAddr{IP: net.ParseIP(outgoingIP.String()), Port: 0}
			} else {
				log.Printf("Connecting to %s [%s] from local net", req.URL.Host, targetIp)
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

	log.Printf("Starting server on  %s:%d", bind, port)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", bind, port), proxy)

	if err != nil {
		log.Fatal(err)
	}

}

func addV6Route(cidr string) {
	delCmd := exec.Command("ip", "route", "del", "local", cidr, "dev", "lo")
	delCmd.Run()

	addCmd := exec.Command("ip", "route", "add", "local", cidr, "dev", "lo")
	if err := addCmd.Run(); err != nil {
		log.Fatalf("Failed to add route: %v", err)
	} else {
		log.Printf("Added route %s dev lo", cidr)
	}
}

func setV6Forwarding() {
	// Enable IPv6 forwarding
	err := exec.Command("sysctl", "-w", "net.ipv6.conf.all.forwarding=1").Run()
	if err != nil {
		log.Fatalf("Failed to enable IPv6 forwarding: %v", err)
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
func isRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("Failed to get current user: %s\n", err)
		return false
	}
	return currentUser.Uid == "0"
}

func (a *AuthConfig) checkAuth(req *http.Request) bool {
	authHeader := req.Header.Get("Proxy-Authorization")
	if authHeader == "" {
		return false
	}

	prefix := "Basic "
	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}

	encoded := strings.TrimPrefix(authHeader, prefix)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}

	return parts[0] == a.username && parts[1] == a.password
}
