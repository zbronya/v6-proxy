package proxy

import (
	"encoding/base64"
	"fmt"
	"github.com/elazarl/goproxy"
	"github.com/zbronya/v6-proxy-pool/config"
	"github.com/zbronya/v6-proxy-pool/netutils"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

func NewProxyServer(cfg config.Config) *goproxy.ProxyHttpServer {

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	proxy.OnRequest().DoFunc(
		func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			username := cfg.AuthConfig.Username
			password := cfg.AuthConfig.Password
			if username != "" && password != "" && !checkAuth(username, password, req) {
				return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusProxyAuthRequired, "Proxy Authentication Required")
			}
			return req, nil
		},
	)

	proxy.OnRequest().HijackConnect(
		func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
			username := cfg.AuthConfig.Username
			password := cfg.AuthConfig.Password
			if username != "" && password != "" && !checkAuth(username, password, req) {
				client.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"Proxy\"\r\n\r\n"))
				client.Close()
				return
			}

			host := req.URL.Hostname()
			targetIp, isV6, err := netutils.GetIPAddress(host)
			if err != nil {
				log.Printf("Get IP address error: %v", err)
				return
			}

			if !isV6 {
				log.Printf("Connecting to %s [%s] from local net", req.URL.Host, targetIp)
				handleDirectConnection(req, client)
			} else {
				outgoingIP, err := netutils.RandomV6(cfg.CIDR)
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
			targetIp, isV6, err := netutils.GetIPAddress(host)
			if err != nil {
				log.Printf("Get IP address error: %v", err)
				return req, nil
			}

			var localAddr *net.TCPAddr

			if isV6 {
				outgoingIP, err := netutils.RandomV6(cfg.CIDR)
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

	return proxy
}

func checkAuth(username string, password string, req *http.Request) bool {
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

	return parts[0] == username && parts[1] == password
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
