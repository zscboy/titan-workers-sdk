package main

import (
	"fmt"
	"net"

	logging "github.com/ipfs/go-log/v2"

	"github.com/zscboy/workerd-sdk/proxy"
	"github.com/zscboy/workerd-sdk/socks5"
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
	logging.SetDebugLogging()
	tunMgr = proxy.NewTunManager(uuid, tunnelCount, tunnelCap, url)
	tunMgr.Startup()

	startSocks5Server("127.0.0.1:8000")
}

func startSocks5Server(address string) {
	conf := &socks5.Config{ConnectHandler: connectHandler}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	// Create SOCKS5 proxy on localhost port 8000
	fmt.Println("listen on ", address)
	if err := server.ListenAndServe("tcp", address); err != nil {
		panic(err)
	}
}
