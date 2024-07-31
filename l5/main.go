package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/zscboy/titan-workers-sdk/config"
	httpproxy "github.com/zscboy/titan-workers-sdk/http"
	"github.com/zscboy/titan-workers-sdk/proxy"
	"github.com/zscboy/titan-workers-sdk/socks5"
)

var log = logging.Logger("l5")

func main() {
	if err := run(os.Args); err != nil {
		log.Errorf(err.Error())
	}

}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Please specify the name of the config file")
	}

	configFilePath := args[1]
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

	tunMgr := proxy.NewTunManager(uuid.NewString(), cfg.Tun.Count, cfg.Tun.Cap, cfg.Tun.URL, cfg.Tun.AuthKey)
	tunMgr.Startup()

	go func() {
		httpProxy := httpproxy.NewProxyServer(cfg.Http.ListenAddress, tunMgr)
		httpProxy.Start()
	}()

	socks5Server := socks5.Socks5Server{OnConnect: tunMgr.OnAcceptRequest}
	socks5Server.StartSocks5Server(cfg.Socks5.ListenAddress)

	return nil
}
