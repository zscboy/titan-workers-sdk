package main

import (
	"fmt"
	"net"

	logging "github.com/ipfs/go-log/v2"
	workerdsdk "github.com/zscboy/workerd-sdk"

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

var tunMgr *workerdsdk.TunMgr

func connectHandler(conn net.Conn, req *socks5.Request) error {
	defer conn.Close()

	fmt.Println("dest addrss ", *req.DestAddr)
	tunMgr.OnAcceptRequest(conn, &workerdsdk.DestAddr{Addr: req.DestAddr.IP.String(), Port: req.DestAddr.Port})

	return nil
}

// type Handler struct {
// 	tunMgr *workerdsdk.TunMgr
// }

// func (h *Handler) TCPHandle(server *socks5.Server, conn *net.TCPConn, req *socks5.Request) error {
// 	fmt.Println("dest addrss ", string(req.DstAddr), string(req.DstPort))
// 	// h.tunMgr.OnAcceptRequest(conn, &workerdsdk.DestAddr{Addr: req.DestAddr.IP.String(), Port: req.DestAddr.Port})
// 	return nil
// }

// func (h *Handler) UDPHandle(*socks5.Server, *net.UDPAddr, *socks5.Datagram) error {
// 	return fmt.Errorf("not implement")
// }

func main() {
	logging.SetDebugLogging()
	tunMgr = workerdsdk.NewTunManager(uuid, tunnelCount, tunnelCap, url)
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

	// server, err := socks5.NewClassicServer(fmt.Sprintf("%s:%d", addr, port), addr, "", "", 0, 0)
	// if err != nil {
	// 	fmt.Println("new socks5 server %s", err.Error())
	// 	return
	// }
	// server.ListenAndServe(handler)
}
