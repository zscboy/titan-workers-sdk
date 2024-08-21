package proxy

import (
	"net"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/zscboy/titan-workers-sdk/selector"
)

var log = logging.Logger("proxy")

const (
	keepaliveIntervel = 3 * time.Second
	sortIntervel      = 3 * time.Second
)

// type Selector interface {
// 	GetServerURL() (string, error)
// 	FindNode(nodeID string) (string, error)
// 	// ReconnectCount()
// }

type DestAddr struct {
	Addr string
	Port int
}

type TunMgr struct {
	// uuid        string
	tunnelCount int
	tunnelCap   int
	selector    selector.TunSelector
	// url            string
	// tunnels        []*Tunnel
	// reconnects     []int
	// sortedTunnels  []*Tunnel
	// currentTunIdex int

	// reconnectsLock sync.Mutex

	// url     string
	// authKey string

	// cancelKeepAlive  chan bool
	// cancelSortTunnel chan bool
	// ctx            context.Context
	// ctxCancel      context.CancelFunc
	// nodes []*Node
	tunPool *TunPool
}

func NewTunManager(tunCount, tunCap int, tunSelector selector.TunSelector) *TunMgr {
	// if len() == 0 {
	// 	panic("url can not empty")
	// }
	tm := &TunMgr{tunnelCount: tunCount, tunnelCap: tunCap, selector: tunSelector}
	return tm

	// return &TunMgr{
	// 	// uuid: uuid,
	// 	// tunnelCount: tunnelCount,
	// 	// tunnelCap:   tunnelCap,
	// 	// reconnects:     make([]int, 0),
	// 	// reconnectsLock: sync.Mutex{},
	// 	// url:              url,
	// 	// authKey:          authKey,
	// 	tunPool: tp,
	// 	// cancelKeepAlive:  make(chan bool),
	// 	// cancelSortTunnel: make(chan bool),
	// }
}

func (tm *TunMgr) Startup() {
	// tm.tunnels = make([]*Tunnel, 0, tm.tunnelCount)
	// tm.sortedTunnels = make([]*Tunnel, 0, tm.tunnelCount)

	// for i := 0; i < len(tm.nodes); i++ {
	// 	node := tm.nodes[i]
	// 	tunnel := newTunnel(tm.uuid, i, tm, node.URL, node.Relays)
	// 	tm.tunnels = append(tm.tunnels, tunnel)
	// 	tm.sortedTunnels = append(tm.sortedTunnels, tunnel)
	// }

	// tm.ctx, tm.ctxCancel = context.WithCancel(context.Background())

	// go tm.keepAlive()
	// go tm.doSortTunnels()
	tunPool := newTunPool(tm.tunnelCount, tm.tunnelCap, tm.selector, tm)
	tunPool.startup()
	tm.tunPool = tunPool
}

func (tm *TunMgr) Reset(tunInfo *selector.TunInfo) error {
	return tm.tunPool.reset(tunInfo)
}

func (tm *TunMgr) OnAcceptRequest(conn net.Conn, dest *DestAddr) {
	// allocate tunnel for sock
	tun := tm.allocTunnelForRequest()
	if tun == nil {
		log.Errorf("[TunMgr] failed to alloc tunnel for sock, discard it")
		return
	}

	log.Debug("OnAcceptRequest alloc tun ", tun.uuid)

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
	// length := len(tm.sortedTunnels)
	// currentIdx := tm.currentTunIdex

	// for i := currentIdx; i < length; i++ {
	// 	tun := tm.sortedTunnels[i]
	// 	if !tun.isConnected() || tun.isFulled() {
	// 		log.Infof("allocTunnelForRequest isConnected %#v, isFulled%#v", tun.isConnected(), tun.isFulled())
	// 		continue
	// 	}

	// 	tm.currentTunIdex = (i + 1) % length
	// 	return tun
	// }

	// for i := 0; i < currentIdx; i++ {
	// 	tun := tm.sortedTunnels[i]
	// 	if !tun.isConnected() || tun.isFulled() {
	// 		log.Infof("allocTunnelForRequest isConnected %#v, isFulled%#v", tun.isConnected(), tun.isFulled())
	// 		continue
	// 	}

	// 	tm.currentTunIdex = (i + 1) % length
	// 	return tun
	// }

	// return nil
	return tm.tunPool.allocTunnelForRequest()
}

// func (tm *TunMgr) doSortTunnels() {
// 	defer log.Infof("doSortTunnels exist")

// 	ticker := time.NewTicker(keepaliveIntervel)

// 	for {
// 		select {
// 		case <-tm.cancelSortTunnel:
// 			return
// 		case <-ticker.C:

// 			sort.Slice(tm.sortedTunnels, func(i, j int) bool {
// 				return tm.sortedTunnels[i].busy < tm.sortedTunnels[j].busy
// 			})

// 			lenght := len(tm.sortedTunnels)
// 			for i := 0; i < lenght; i++ {
// 				tun := tm.sortedTunnels[i]
// 				tun.resetBusy()
// 			}

// 			tm.currentTunIdex = 0
// 		}
// 	}

// }

// func (tm *TunMgr) keepAlive() {
// 	defer log.Infof("keepAlive exist")

// 	ticker := time.NewTicker(keepaliveIntervel)
// 	// log.Infof("start keepAlive,intervel %d", keepaliveIntervel)./ti
// 	for {
// 		select {
// 		case <-tm.cancelKeepAlive:
// 			return
// 		case <-ticker.C:
// 			length := len(tm.tunnels)
// 			// log.Infof("ticker, tunnels len:%d", length)
// 			for i := 0; i < length; i++ {
// 				tun := tm.tunnels[i]
// 				if !tun.isConnected() || tun.isDestroy {
// 					continue
// 				}
// 				tun.sendPing()
// 			}

// 			tm.reconnectsLock.Lock()
// 			reconnects := append([]int{}, tm.reconnects...)
// 			tm.reconnects = make([]int, 0)
// 			tm.reconnectsLock.Unlock()

// 			for _, idx := range reconnects {
// 				tun := tm.tunnels[idx]
// 				if tun.isDestroy {
// 					log.Infof("tun %d is destry, not need to reconnect", tun.idx)
// 					continue
// 				}
// 				if tun.isConnected() {
// 					log.Infof("tun %d is connected, not need to reconnect", tun.idx)
// 					continue
// 				}

// 				// tm.reconnectCount()
// 				if err := tun.reconnect(); err != nil {
// 					log.Errorf("reconnect failed %s", err.Error())
// 				}
// 			}
// 		}
// 	}
// }

func (tm *TunMgr) onTunnelBroken(tun *Tunnel) {
	tm.tunPool.onTunnelBroken(tun)
}
