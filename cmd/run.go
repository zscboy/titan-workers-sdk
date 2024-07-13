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

	// selector, err := newSampleSelector("ws://localhost:2345/project/e_23a42fb9-0ce7-43bb-9afd-af23c576dabc/tun")
	selector, err := newCustomSelector(cfg)
	if err != nil {
		return fmt.Errorf("newCustomSelector " + err.Error())
	}

	url, err := selector.GetNodeURL()
	if err != nil {
		return fmt.Errorf("newCustomSelector " + err.Error())
	}

	tunMgr := proxy.NewTunManager(uuid.NewString(), cfg.Tun.Count, cfg.Tun.Cap, url)
	tunMgr.Startup()

	go func() {
		httpProxy := httpproxy.NewProxyServer(cfg.Http.ListenAddress, tunMgr)
		httpProxy.Start()
	}()

	go func() {
		localHttpServer := LocalHttpServer{address: cfg.LocalHttpServer.ListenAddress, tunMgr: tunMgr, selector: selector}
		localHttpServer.start()
	}()

	socks5Server := socks5Server{tunMgr: tunMgr}
	socks5Server.startSocks5Server(cfg.Socks5.ListenAddress)

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
