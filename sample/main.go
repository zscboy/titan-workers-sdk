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
	socks5Port  = 1080
	httpPort    = 1081
	uuid        = "ee80e87b-fc41-4e59-a722-7c3fee039cb4"
	tunnelCount = 1
	tunnelCap   = 100
	// url         = "wss://4a6a158a-3788-48b7-a0ab-f936afd778c2.test.titannet.io:2345/project/e_85a7e089-0ce4-4337-94ca-763587a07f45/3455465e-02b6-49de-b326-773a7fe5f424/tun"
	url = "wss://4a6a158a-3788-48b7-a0ab-f936afd778c2.test.titannet.io:2345/project/e_85a7e089-0ce4-4337-94ca-763587a07f45/3455465e-02b6-49de-b326-773a7fe5f424/tun"
)

var tunMgr *proxy.TunMgr

func connectHandler(conn net.Conn, req *socks5.Request) error {
	defer conn.Close()

	fmt.Println("on socks5 connect ", *req.DestAddr)
	addr := req.DestAddr.IP.String()
	if len(req.DestAddr.FQDN) > 0 {
		addr = req.DestAddr.FQDN
	}
	tunMgr.OnAcceptRequest(conn, &proxy.DestAddr{Addr: addr, Port: req.DestAddr.Port})

	return nil
}

func main() {
	startProxy()
}

func startProxy() {
	logging.SetDebugLogging()
	tunMgr = proxy.NewTunManager(uuid, tunnelCount, tunnelCap, &customAccessPoint{ServerURL: url})
	tunMgr.Startup()

	go func() {
		httpProxy := http.NewProxyServer(fmt.Sprintf(":%d", httpPort), tunMgr)
		httpProxy.Start()
	}()

	startSocks5Server(fmt.Sprintf("localhost:%d", socks5Port))
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

type customAccessPoint struct {
	ServerURL string
}

func (ap *customAccessPoint) GetServerURL() (string, error) {
	return ap.ServerURL, nil
}
func (ap *customAccessPoint) RefreshServerURL() {

}
