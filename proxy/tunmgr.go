package proxy

import (
	"net"
	"sort"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("workerdsdk")

const (
	keepaliveIntervel = 15 * time.Second
	sortIntervel      = 3 * time.Second
)

type DestAddr struct {
	Addr string
	Port int
}

type TunMgr struct {
	uuid           string
	tunnelCount    int
	tunnelCap      int
	url            string
	tunnels        []*Tunnel
	reconnects     []int
	sortedTunnels  []*Tunnel
	currentTunIdex int
}

func NewTunManager(uuid string, tunnelCount, tunnelCap int, url string) *TunMgr {
	return &TunMgr{uuid: uuid, tunnelCount: tunnelCount, tunnelCap: tunnelCap, url: url, reconnects: make([]int, 0)}
}

func (tm *TunMgr) Startup() {
	tm.tunnels = make([]*Tunnel, 0, tm.tunnelCount)
	tm.sortedTunnels = make([]*Tunnel, 0, tm.tunnelCount)

	for i := 0; i < tm.tunnelCount; i++ {
		tunnel := newTunnel(tm.uuid, i, tm, tm.url, tm.tunnelCap)
		tm.tunnels = append(tm.tunnels, tunnel)
		tm.sortedTunnels = append(tm.sortedTunnels, tunnel)

		go func() {
			if err := tunnel.connect(); err != nil {
				log.Errorf("connect %s", err.Error())
			}
		}()
	}

	go tm.keepAlive()

	go tm.doSortTunnels()
}

func (tm *TunMgr) OnAcceptRequest(conn net.Conn, dest *DestAddr) {
	// allocate tunnel for sock
	tun := tm.allocTunnelForRequest()
	if tun == nil {
		log.Errorf("[TunMgr] failed to alloc tunnel for sock, discard it")
		return
	}

	if err := tun.onAcceptRequest(conn, dest); err != nil {
		log.Errorf("onAcceptRequest %s", err.Error())
	}
}

func (t *Tunnel) onAcceptHTTPsRequest(conn net.Conn, dest *DestAddr, header []byte) error {
	req, err := t.acceptRequestInternal(conn, dest)
	if err != nil {
		return err
	}

	_, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n" +
		"Proxy-agent: linproxy\r\n" +
		"\r\n"))
	if err != nil {
		return err
	}

	return t.onClientRecvData(req.idx, req.tag, header)
}

func (t *Tunnel) onAcceptHTTPRequest(conn net.Conn, dest *DestAddr, header []byte) error {
	log.Infof("onAcceptHTTPRequest, dest addr %s port %d", dest.Addr, dest.Port)
	req, err := t.acceptRequestInternal(conn, dest)
	if err != nil {
		return err
	}

	err = t.serveHTTPRequest(conn, req.idx, req.tag)
	if err != nil {
		return err
	}

	return t.onClientRecvData(req.idx, req.tag, header)
}

func (tm *TunMgr) allocTunnelForRequest() *Tunnel {
	length := len(tm.sortedTunnels)
	currentIdx := tm.currentTunIdex

	for i := currentIdx; i < length; i++ {
		tun := tm.sortedTunnels[i]
		if !tun.isConnected() || tun.isFulled() {
			log.Infof("allocTunnelForRequest isConnected %#v, isFulled%#v", tun.isConnected(), tun.isFulled())
			continue
		}

		tm.currentTunIdex = (i + 1) % length
		return tun
	}

	for i := 0; i < currentIdx; i++ {
		tun := tm.sortedTunnels[i]
		if !tun.isConnected() || tun.isFulled() {
			log.Infof("allocTunnelForRequest isConnected %#v, isFulled%#v", tun.isConnected(), tun.isFulled())
			continue
		}

		tm.currentTunIdex = (i + 1) % length
		return tun
	}

	return nil
}

func (tm *TunMgr) doSortTunnels() {
	ticker := time.NewTicker(keepaliveIntervel)

	for {
		<-ticker.C

		sort.Slice(tm.sortedTunnels, func(i, j int) bool {
			return tm.sortedTunnels[i].busy < tm.sortedTunnels[j].busy
		})

		lenght := len(tm.sortedTunnels)
		for i := 0; i < lenght; i++ {
			tun := tm.sortedTunnels[i]
			tun.resetBusy()
		}

		tm.currentTunIdex = 0
	}

}

func (tm *TunMgr) keepAlive() {
	ticker := time.NewTicker(keepaliveIntervel)

	for {
		<-ticker.C

		length := len(tm.tunnels)
		for i := 0; i < length; i++ {
			tun := tm.tunnels[i]
			if !tun.isConnected() {
				continue
			}
			tun.sendPing()
		}

		reconnects := tm.reconnects
		tm.reconnects = make([]int, 0)

		length = len(reconnects)
		for i := 0; i < length; i++ {
			idx := reconnects[i]
			tun := tm.tunnels[idx]
			if tun.isConnected() {
				continue
			}
			go tun.connect()
		}
	}
}

func (tm *TunMgr) onTunnelBroken(tun *Tunnel) {
	tm.reconnects = append(tm.reconnects, tun.idx)
}
