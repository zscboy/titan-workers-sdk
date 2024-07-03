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

	// selector, err := newSampleSelector("ws://47.76.157.173:2345/project/e_af22c923-c262-4c37-958b-be1479815672/4457f380-105a-4757-8b26-4184d8afc30a/tun")
	selector, err := newCustomSelector(cfg)
	if err != nil {
		return fmt.Errorf("newCustomSelector " + err.Error())
	}

	url, err := selector.GetServerURL()
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

	url, err := local.selector.FindNode(nodeID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	local.tunMgr.ResetTunnel(url)
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

func (local *LocalHttpServer) start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/change", local.chnodeHandler)
	mux.HandleFunc("/ls", local.lsnodeHandler)
	mux.HandleFunc("/query", local.lsnodeHandler)

	log.Infof("Starting local http server on %s", local.address)
	err := http.ListenAndServe(local.address, mux)
	if err != nil {
		log.Infof("Error starting local server:", err)
	}
}
