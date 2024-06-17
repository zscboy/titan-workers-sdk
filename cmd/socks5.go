package main

import (
	"net"

	socks5 "github.com/zscboy/go-socks5"
	"github.com/zscboy/titan-workers-sdk/proxy"
)

type socks5Server struct {
	tunMgr *proxy.TunMgr
}

func (s *socks5Server) startSocks5Server(address string) {
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

func (s *socks5Server) connectHandler(conn net.Conn, req *socks5.Request) error {
	defer func() {
		conn.Close()
		log.Debugf("close socks5 connect %s", *req.DestAddr)
	}()

	log.Debugf("on socks5 connect %s", *req.DestAddr)

	addr := req.DestAddr.IP.String()
	if len(req.DestAddr.FQDN) > 0 {
		addr = req.DestAddr.FQDN
	}
	s.tunMgr.OnAcceptRequest(conn, &proxy.DestAddr{Addr: addr, Port: req.DestAddr.Port})

	return nil
}
