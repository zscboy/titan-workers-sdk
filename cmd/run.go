package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/zscboy/titan-workers-sdk/config"
	httpproxy "github.com/zscboy/titan-workers-sdk/http"
	"github.com/zscboy/titan-workers-sdk/proxy"
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

	// selector, err := newSampleSelector("ws://localhost:4000/tun")
	// selector, err := newSampleSelector("wss://8cc13918-0380-4b80-8d4d-f385cd577cba.cassini-l1.titannet.io:2345/project/e_33536f04-142b-43db-ae50-06498cc9b8f9/4fd2d416-35cb-4c55-9faf-bee933094315/tun")
	// selector, err := newSampleSelector("wss://5d284569-1a86-40f7-9887-5aff27cd1cbb.test.titannet.io:2345/project/e_85a7e089-0ce4-4337-94ca-763587a07f45/f33939a6-b7d8-4bd1-a189-53f5099f8b98/tun")
	selector, err := newCustomSelector(cfg)
	if err != nil {
		return fmt.Errorf("newCustomSelector " + err.Error())
	}

	url, err := selector.GetNodeURL()
	if err != nil {
		return fmt.Errorf("newCustomSelector " + err.Error())
	}

	tunMgr := proxy.NewTunManager(uuid.NewString(), cfg.Tun.Count, cfg.Tun.Cap, url, "")
	tunMgr.Startup()

	go func() {
		httpProxy := httpproxy.NewProxyServer(cfg.Http.ListenAddress, tunMgr)
		httpProxy.Start()
	}()

	go func() {
		localHttpServer := LocalHttpServer{address: cfg.LocalHttpServer.ListenAddress, tunMgr: tunMgr, selector: selector}
		localHttpServer.start()
	}()

	socks5Server := socks5.Socks5Server{OnConnect: tunMgr.OnAcceptRequest}
	socks5Server.StartSocks5Server(cfg.Socks5.ListenAddress)

	return nil
}

type LocalHttpServer struct {
	address  string
	tunMgr   *proxy.TunMgr
	selector *customSelector
}

func (local *LocalHttpServer) chnodeHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("id")
	if len(nodeID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("id can not empty"))
		return
	}
	log.Infof("chnodeHandler id %d", nodeID)
	if local.selector.currentUseNode != nil {
		if local.selector.currentUseNode.ID == nodeID {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("node is in use"))
			return
		}
	}

	url, err := local.selector.FindNode(nodeID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	local.tunMgr.RestartWith(url)
}

func (local *LocalHttpServer) lsnodeHandler(w http.ResponseWriter, r *http.Request) {
	buf, err := json.Marshal(local.selector.pInfos)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(buf)
}

func (local *LocalHttpServer) queryHandler(w http.ResponseWriter, r *http.Request) {
	node := local.selector.CurrentNode()
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
	node := local.selector.CurrentNode()
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

	w := web.NewWeb(local.selector.pInfos, local.selector.currentUseNode)
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
