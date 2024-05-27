package proxy

import (
	"net"
	"sort"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("proxy")

const (
	keepaliveIntervel = 3 * time.Second
	sortIntervel      = 3 * time.Second
)

type DestAddr struct {
	Addr string
	Port int
}

type TunMgr struct {
	uuid        string
	tunnelCount int
	tunnelCap   int
	// url            string
	tunnels        []*Tunnel
	reconnects     []int
	sortedTunnels  []*Tunnel
	currentTunIdex int

	reconnectsLock sync.Mutex

	accessPoint AccessPoint
}

func NewTunManager(uuid string, tunnelCount, tunnelCap int, ap AccessPoint) *TunMgr {
	if ap == nil {
		panic("access point can not empty")
	}

	return &TunMgr{
		uuid:           uuid,
		tunnelCount:    tunnelCount,
		tunnelCap:      tunnelCap,
		reconnects:     make([]int, 0),
		reconnectsLock: sync.Mutex{},
		accessPoint:    ap,
	}
}

func (tm *TunMgr) Startup() {
	tm.tunnels = make([]*Tunnel, 0, tm.tunnelCount)
	tm.sortedTunnels = make([]*Tunnel, 0, tm.tunnelCount)

	for i := 0; i < tm.tunnelCount; i++ {
		tunnel := newTunnel(tm.uuid, i, tm, tm.tunnelCap)
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

func (tm *TunMgr) OnAcceptHTTPsRequest(conn net.Conn, dest *DestAddr) {
	// allocate tunnel for sock
	tun := tm.allocTunnelForRequest()
	if tun == nil {
		log.Errorf("[TunMgr] failed to alloc tunnel for sock, discard it")
		return
	}

	if err := tun.onAcceptHTTPsRequest(conn, dest); err != nil {
		log.Errorf("onAcceptHTTPRequest %s", err.Error())
	}
}

func (tm *TunMgr) OnAcceptHTTPRequest(conn net.Conn, dest *DestAddr, header []byte) {
	// allocate tunnel for sock
	tun := tm.allocTunnelForRequest()
	if tun == nil {
		log.Errorf("[TunMgr] failed to alloc tunnel for sock, discard it")
		return
	}

	if err := tun.onAcceptHTTPRequest(conn, dest, header); err != nil {
		log.Errorf("onAcceptHTTPRequest %s", err.Error())
	}
}

func (t *Tunnel) onAcceptHTTPsRequest(conn net.Conn, dest *DestAddr) error {
	log.Infof("onAcceptHTTPsRequest, dest addr %s port %d", dest.Addr, dest.Port)

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

	return t.serveConn(conn, req.idx, req.tag)
}

func (t *Tunnel) onAcceptHTTPRequest(conn net.Conn, dest *DestAddr, header []byte) error {
	log.Infof("onAcceptHTTPRequest, dest addr %s port %d", dest.Addr, dest.Port)
	req, err := t.acceptRequestInternal(conn, dest)
	if err != nil {
		return err
	}

	defer t.onClientRecvFinished(req.idx, req.tag)

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

		tm.reconnectsLock.Lock()
		reconnects := append([]int{}, tm.reconnects...)
		tm.reconnects = make([]int, 0)
		tm.reconnectsLock.Unlock()

		for _, idx := range reconnects {
			tun := tm.tunnels[idx]
			if tun.isConnected() {
				log.Infof("tun %s is connected, not need to reconnect", tun.idx)
				continue
			}

			go func() {
				if err := tun.connect(); err != nil {
					log.Errorf("reconnect failed %s", err.Error())
				}
			}()
		}
	}
}

func (tm *TunMgr) onTunnelBroken(tun *Tunnel) {
	tm.reconnectsLock.Lock()
	defer tm.reconnectsLock.Unlock()
	tm.reconnects = append(tm.reconnects, tun.idx)
}

func (tm *TunMgr) getServerURL() (string, error) {
	return tm.accessPoint.GetServerURL()
}
