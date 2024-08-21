package proxy

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zscboy/titan-workers-sdk/selector"
)

type TunPool struct {
	tunnels  []*Tunnel
	tunLock  sync.Mutex
	tunCount int
	tunCap   int

	tunSelector selector.TunSelector
	tm          *TunMgr
}

func newTunPool(tunCount, tunCap int, tunSelector selector.TunSelector, tm *TunMgr) *TunPool {
	return &TunPool{tunCount: tunCount, tunCap: tunCap, tunLock: sync.Mutex{}, tunSelector: tunSelector, tm: tm}
}

func (tp *TunPool) startup() {
	tunInfos := tp.tunSelector.GetTunInfos(tp.tunCount)
	if len(tunInfos) == 0 {
		panic("Can not get tun")
	}
	for _, tunInfo := range tunInfos {
		tunnel, err := newTunnel(uuid.NewString(), tp.tm, tp.tunCap, tunInfo)
		if err != nil {
			log.Errorf("new Tunnel failed %s", err.Error())
			continue
		}
		tp.tunnels = append(tp.tunnels, tunnel)
	}

	if len(tp.tunnels) == 0 {
		panic("No available tun exist")
	}

	log.Infof("startup tunnels len %d", len(tp.tunnels))

	go tp.keepalive()
	// go tp.sortTunnels()
	go tp.refresh()

}

func (tp *TunPool) allocTunnelForRequest() *Tunnel {
	tp.tunLock.Lock()
	defer tp.tunLock.Unlock()

	if len(tp.tunnels) > 0 {
		return tp.tunnels[0]
	}
	return nil
}

func (tp *TunPool) addTunnels() {

	tunInfos := tp.tunSelector.GetTunInfos(tp.tunCount)
	// filter tun
	tp.tunLock.Lock()
	defer tp.tunLock.Unlock()

	tunMap := make(map[string]*Tunnel)
	for _, tun := range tp.tunnels {
		tunMap[tun.targetNodeID] = tun
	}

	availableTunInfos := make([]*selector.TunInfo, 0)
	for _, tunInfo := range tunInfos {
		if _, ok := tunMap[tunInfo.NodeID]; !ok {
			availableTunInfos = append(availableTunInfos, tunInfo)
		}
	}

	log.Debugf("filter availableTunInfos len %d, tunInfos len %d, tun len %d", len(availableTunInfos), len(tunInfos), len(tp.tunnels))
	for _, tunInfo := range availableTunInfos {
		tunnel, err := newTunnel(uuid.NewString(), tp.tm, tp.tunCap, tunInfo)
		if err != nil {
			log.Errorf("New tunnel failed %s", err.Error())
			continue
		}

		tp.tunnels = append(tp.tunnels, tunnel)
		if len(tp.tunnels) >= tp.tunCount {
			break
		}
	}
}

func (tp *TunPool) rmeoveTunnel(tun *Tunnel) {
	tp.tunLock.Lock()
	defer tp.tunLock.Unlock()

	for i, t := range tp.tunnels {
		if t.uuid == tun.uuid {
			tp.tunnels = append(tp.tunnels[:i], tp.tunnels[i+1:]...)
		}
	}
}

func (tp *TunPool) onTunnelBroken(tun *Tunnel) {
	log.Infof("onTunnelBroken %s", tun.uuid)
	tp.rmeoveTunnel(tun)
}

func (tp *TunPool) refresh() {
	ticker := time.NewTicker(keepaliveIntervel)
	for {
		<-ticker.C
		if len(tp.tunnels) < tp.tunCount {
			tp.addTunnels()
		}
	}
}

func (tp *TunPool) keepalive() {
	ticker := time.NewTicker(keepaliveIntervel)

	for {
		<-ticker.C
		for _, t := range tp.tunnels {
			t.sendPing()
		}

		// Disposal of dead tunnels
		for _, t := range tp.tunnels {
			if time.Since(t.lastPongTime) > 3*keepaliveIntervel {
				tp.onTunnelBroken(t)
			}
		}
	}
}

func (tp *TunPool) sortTunnels() {
	ticker := time.NewTicker(sortIntervel)

	for {
		<-ticker.C

		sort.Slice(tp.tunnels, func(i, j int) bool {
			return delayAverage(tp.tunnels[i].delays) < delayAverage(tp.tunnels[j].delays)
		})
	}

}

func delayAverage(delays []int) int {
	if len(delays) == 0 {
		return math.MaxInt
	}

	count := 0
	for _, delay := range delays {
		count += delay
	}

	return count / len(delays)
}

func (tp *TunPool) reset(tunInfo *selector.TunInfo) error {
	for _, t := range tp.tunnels {
		tp.rmeoveTunnel(t)
	}

	tunnel, err := newTunnel(uuid.NewString(), tp.tm, tp.tunCap, tunInfo)
	if err != nil {
		return err
	}
	tp.tunnels = append(tp.tunnels, tunnel)

	return nil
}
