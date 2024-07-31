package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	worker "github.com/zscboy/titan-workers-sdk"
	"github.com/zscboy/titan-workers-sdk/config"
)

var listTunnelsCmd = &cobra.Command{
	Use:     "tuns",
	Short:   "get tunnels",
	Example: "tuns --project-id you-project-id ./config.toml",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Args can not empty")
			return
		}

		configFilePath := args[0]
		if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
			fmt.Println("Config file %s not exist", configFilePath)
			return
		}

		cfg, err := config.ParseConfig(configFilePath)
		if err != nil {
			fmt.Println("Parse config file: %s", err.Error())
			return
		}

		projectID, err := cmd.Flags().GetString("project-id")
		if len(projectID) == 0 || err != nil {
			fmt.Println("Must set --project-id")
			return
		}

		// fmt.Println("config ", *cfg)
		// fmt.Println("nodes ", nodes)

		wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
		w, err := worker.NewWorker(wConfig)
		if err != nil {
			fmt.Println("NewWorker %s", err.Error())
			return
		}

		tuns, err := w.GetTunnels(projectID)
		if err != nil {
			fmt.Println("NewWorker %s", err.Error())
			return
		}

		if len(tuns) == 0 {
			fmt.Println("len(tuns) == 0")
			return
		}

		fmt.Println("NodeID:", tuns[0].NodeID)
		fmt.Println("WSURL:", tuns[0].WSURL)
	},
}
