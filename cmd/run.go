package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/zscboy/titan-workers-sdk/config"
	"github.com/zscboy/titan-workers-sdk/http"
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

	selector, err := newCustomSelector(cfg)
	if err != nil {
		return fmt.Errorf("newSampleSelector " + err.Error())
	}

	tunMgr := proxy.NewTunManager(uuid.NewString(), cfg.Tun.Count, cfg.Tun.Cap, selector)
	tunMgr.Startup()

	go func() {
		httpProxy := http.NewProxyServer(cfg.Http.ListenAddress, tunMgr)
		httpProxy.Start()
	}()

	socks5Server := socks5Server{tunMgr: tunMgr}
	socks5Server.startSocks5Server(cfg.Socks5.ListenAddress)

	return nil
}
