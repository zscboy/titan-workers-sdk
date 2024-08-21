package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	worker "github.com/zscboy/titan-workers-sdk"
	"github.com/zscboy/titan-workers-sdk/config"
	"github.com/zscboy/titan-workers-sdk/proxy"
	"github.com/zscboy/titan-workers-sdk/selector"
	"github.com/zscboy/titan-workers-sdk/socks5"
	"github.com/zscboy/titan-workers-sdk/web"
)

func run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Please specify the name of the config file")
	}

	configFilePath := args[0]
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist.\n", configFilePath)
	}

	cfg, err := config.ParseConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("parse config error " + err.Error())
	}

	logLevel := logging.LevelInfo
	if cfg.Log.Level == "debug" {
		logLevel = logging.LevelDebug
	}
	logging.SetAllLoggers(logLevel)

	wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return err
	}

	pInfos, err := loadProjects(w)
	if err != nil {
		return err
	}

	node := filterNodeOrFirst(cfg.Selector.DefaultNodeID, pInfos)
	// copy a value from node, can not refrence to node
	currentNode := &worker.Node{ID: node.ID, URL: node.URL, Status: node.Status, AreaID: node.AreaID, IP: node.IP}

	var ts selector.TunSelector
	if cfg.Selector.Type == selector.TypeAuto {
		ts, err = selector.NewAutoSelector(w, cfg.Selector.AreaID)
		if err != nil {
			return err
		}
	} else if cfg.Selector.Type == selector.TypeFix {
		ts = selector.NewWebSelector(currentNode, pInfos)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Selector type %s not found", cfg.Selector.Type)
	}

	// tunInfos := []*selector.TunInfo{{URL: cfg.Tun.URL, Auth: cfg.Tun.AuthKey}}
	tunMgr := proxy.NewTunManager(cfg.Tun.Count, cfg.Tun.Cap, ts)
	tunMgr.Startup()

	// go func() {
	// 	httpProxy := httpproxy.NewProxyServer(cfg.Http.ListenAddress, tunMgr)
	// 	httpProxy.Start()
	// }()

	go func() {
		localHttpServer := LocalHttpServer{
			address:        cfg.LocalHttpServer.ListenAddress,
			tunMgr:         tunMgr,
			pInfos:         pInfos,
			currentUseNode: currentNode,
		}
		localHttpServer.start()
	}()

	socks5Server := socks5.Socks5Server{OnConnect: tunMgr.OnAcceptRequest}
	socks5Server.StartSocks5Server(cfg.Socks5.ListenAddress)

	return nil
}

func loadProjects(w worker.Worker) ([]*worker.PorjectInfo, error) {
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

	return pInfos, nil
}

func filterNodeOrFirst(nodeID string, pInfos []*worker.PorjectInfo) *worker.Node {
	for _, pInfo := range pInfos {
		for _, node := range pInfo.Nodes {
			if node.ID == nodeID {
				return node
			}
		}
	}

	for _, pInfo := range pInfos {
		for _, node := range pInfo.Nodes {
			return node
		}
	}

	return nil
}

type LocalHttpServer struct {
	address string
	tunMgr  *proxy.TunMgr
	// selector *selector.WebSelector
	pInfos         []*worker.PorjectInfo
	currentUseNode *worker.Node
}

func (local *LocalHttpServer) chnodeHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("id")
	if len(nodeID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("id can not empty"))
		return
	}
	log.Infof("chnodeHandler id %d", nodeID)
	if local.currentUseNode != nil && local.currentUseNode.ID == nodeID {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("node is in use"))
		return
	}

	node, tunInfo := local.findTunInfo(nodeID)
	if tunInfo == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Can not find node %s", nodeID)))
		return
	}

	err := local.tunMgr.Reset(tunInfo)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Set node failed %s", err.Error())))
		return
	}

	*(local.currentUseNode) = node
}

func (local *LocalHttpServer) findTunInfo(nodeID string) (worker.Node, *selector.TunInfo) {
	for _, pInfo := range local.pInfos {
		for _, node := range pInfo.Nodes {
			if node.ID == nodeID {
				url := fmt.Sprintf("%s/project/%s/%s/tun", node.URL, node.ID, pInfo.ID)

				return *node, &selector.TunInfo{NodeID: node.ID, URL: url}
			}
		}
	}
	return worker.Node{}, nil
}

func (local *LocalHttpServer) lsnodeHandler(w http.ResponseWriter, r *http.Request) {
	buf, err := json.Marshal(local.pInfos)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(buf)
}

func (local *LocalHttpServer) queryHandler(w http.ResponseWriter, r *http.Request) {
	node := local.currentUseNode
	if node == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("no node exist"))
		return
	}

	buf, err := json.Marshal(node)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte(string(buf)))
}

func (local *LocalHttpServer) webHandler(w http.ResponseWriter, r *http.Request) {
	node := local.currentUseNode
	if node == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("no node exist"))
		return
	}

	buf, err := json.Marshal(node)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte(string(buf)))
}

func (local *LocalHttpServer) start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/change", local.chnodeHandler)
	mux.HandleFunc("/ls", local.lsnodeHandler)
	mux.HandleFunc("/query", local.queryHandler)

	w := web.NewWeb(local.pInfos, local.currentUseNode)
	mux.HandleFunc("/web", w.WebHandler)
	mux.HandleFunc("/web/getCountryOptions", w.GetCountryOptions)
	mux.HandleFunc("/web/getNodeOptions", w.GetNodeOptions)
	mux.HandleFunc("/web/submit", w.Submit)

	staticServer := http.FileServer(http.FS(web.StaticFiles))
	mux.Handle("/", staticServer)

	log.Infof("Starting local web server on %s", local.address)
	err := http.ListenAndServe(local.address, mux)
	if err != nil {
		log.Infof("Error starting local server:", err)
	}
}
