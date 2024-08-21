package selector

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	worker "github.com/zscboy/titan-workers-sdk"
)

type AutoSelector struct {
	worker       worker.Worker
	projectInfos []*worker.PorjectInfo
	areaID       string
}

func NewAutoSelector(w worker.Worker, areaID string) (*AutoSelector, error) {
	page := 0
	size := 50
	pInfos := make([]*worker.PorjectInfo, 0)
	for {
		projectInfos, err := w.LoadProjects(page, size)
		if err != nil {
			return nil, err
		}

		pInfos = append(pInfos, projectInfos...)
		if len(projectInfos) < size {
			break
		}
		page++
	}

	return &AutoSelector{worker: w, projectInfos: pInfos, areaID: areaID}, nil
}

func (as *AutoSelector) GetTunInfos(count int) []*TunInfo {
	tunInfos := make([]*TunInfo, 0)
	for _, projectInfo := range as.projectInfos {
		for _, node := range projectInfo.Nodes {
			areaID := strings.ToLower(node.AreaID)
			if !strings.Contains(areaID, strings.ToLower(as.areaID)) {
				continue
			}
			url := fmt.Sprintf("%s/project/%s/%s/tun", node.URL, node.ID, projectInfo.ID)
			tunInfos = append(tunInfos, &TunInfo{NodeID: node.ID, URL: url})
		}
	}

	usableTunInfos := make([]*TunInfo, 0)
	for i := 0; i < len(tunInfos); i = i + count {
		end := i + count
		if end > len(tunInfos) {
			end = len(tunInfos)
		}

		connectivitys := checkTunnelsConnectivity(tunInfos[i:end])
		for _, conconnectivity := range connectivitys {
			if conconnectivity.OK {
				usableTunInfos = append(usableTunInfos, &conconnectivity.TunInfo)
			}
		}

		if len(usableTunInfos) >= count {
			return usableTunInfos
		}
	}
	return usableTunInfos
}

type Connectivitys struct {
	OK bool
	TunInfo
}

func checkTunnelsConnectivity(tunInfos []*TunInfo) []*Connectivitys {
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	connectivitys := make([]*Connectivitys, 0, len(tunInfos))

	for _, tunInfo := range tunInfos {
		info := TunInfo{NodeID: tunInfo.NodeID, URL: tunInfo.URL, Relays: tunInfo.Relays, Auth: tunInfo.Auth}
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := checkTunnelConnectivity(&info)
			if err != nil {
				log.Warnf("checkTunnelConnectivity %s, ", err.Error())
				// handler err
			}

			lock.Lock()
			connectivitys = append(connectivitys, &Connectivitys{OK: ok, TunInfo: info})
			lock.Unlock()
		}()
	}

	wg.Wait()
	return connectivitys
}

func checkTunnelConnectivity(tunInfo *TunInfo) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	header := make(http.Header)
	header.Set("User-Timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	if len(tunInfo.Relays) > 0 {
		for _, relay := range tunInfo.Relays {
			header.Add("Relay", relay)
		}
	}

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, tunInfo.URL, header)
	if err != nil {
		if resp != nil {
			buf, _ := io.ReadAll(resp.Body)
			log.Infof("dial err %s, %s %s", err.Error(), string(buf), tunInfo.URL)
		} else {
			log.Infof("dial err %s %s", err.Error(), tunInfo.URL)
		}
		return false, err
	}
	defer conn.Close()

	return true, nil
}
