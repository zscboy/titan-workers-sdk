package socks5

import (
	"net"

	logging "github.com/ipfs/go-log/v2"
	socks5 "github.com/zscboy/go-socks5"
	"github.com/zscboy/titan-workers-sdk/proxy"
)

var log = logging.Logger("socks5")

type Socks5Server struct {
	OnConnect func(conn net.Conn, dest *proxy.DestAddr)
}

func (s *Socks5Server) StartSocks5Server(address string) {
	conf := &socks5.Config{CustomConnectHandler: s.connectHandler}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	// Create SOCKS5 proxy on localhost port 8000
	log.Info("socks5 listen on ", address)
	if err := server.ListenAndServe("tcp", address); err != nil {
		panic(err)
	}
}

func (s *Socks5Server) connectHandler(conn net.Conn, req *socks5.Request) error {
	defer func() {
		conn.Close()
		log.Debugf("close socks5 connect %s", *req.DestAddr)
	}()

	log.Debugf("on socks5 connect %s", *req.DestAddr)

	addr := req.DestAddr.IP.String()
	if len(req.DestAddr.FQDN) > 0 {
		addr = req.DestAddr.FQDN
	}
	s.OnConnect(conn, &proxy.DestAddr{Addr: addr, Port: req.DestAddr.Port})

	return nil
}
