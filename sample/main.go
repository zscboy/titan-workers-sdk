package main

import (
	"fmt"
	"net"

	"github.com/zscboy/titan-workers-sdk/http"

	logging "github.com/ipfs/go-log/v2"

	socks5 "github.com/zscboy/go-socks5"
	"github.com/zscboy/titan-workers-sdk/proxy"
	// "github.com/txthinking/socks5"
)

const (
	socks5Port  = 8000
	httpPort    = 8001
	uuid        = "ee80e87b-fc41-4e59-a722-7c3fee039cb4"
	tunnelCount = 10
	tunnelCap   = 100
	url         = "ws://localhost:8020/tun"
)

var tunMgr *proxy.TunMgr

func connectHandler(conn net.Conn, req *socks5.Request) error {
	defer conn.Close()

	fmt.Println("dest addrss ", *req.DestAddr)
	tunMgr.OnAcceptRequest(conn, &proxy.DestAddr{Addr: req.DestAddr.IP.String(), Port: req.DestAddr.Port})

	return nil
}

func main() {
	startProxy()
}

func startProxy() {
	logging.SetDebugLogging()
	tunMgr = proxy.NewTunManager(uuid, tunnelCount, tunnelCap, url)
	tunMgr.Startup()

	go func() {
		httpProxy := http.NewProxyServer(fmt.Sprintf(":%d", httpPort), tunMgr)
		httpProxy.Start()
	}()

	startSocks5Server("127.0.0.1:8000")
}

func startSocks5Server(address string) {
	conf := &socks5.Config{CustomConnectHandler: connectHandler}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	// Create SOCKS5 proxy on localhost port 8000
	fmt.Println("socks5 listen on ", address)
	if err := server.ListenAndServe("tcp", address); err != nil {
		panic(err)
	}
}
