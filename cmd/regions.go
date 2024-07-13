package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	worker "github.com/zscboy/titan-workers-sdk"
	"github.com/zscboy/titan-workers-sdk/config"
	"github.com/zscboy/titan-workers-sdk/tablewriter"
)

func getRegions(cmd *cobra.Command, args []string) (*worker.AreaList, error) {
	area, err := cmd.Flags().GetString("area")
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("Please specify the name of the config file")
	}

	configFilePath := args[0]
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist.", configFilePath)
	}

	cfg, err := config.ParseConfig(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("parse config error %s", err.Error())
	}

	wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return nil, fmt.Errorf("NewWorker %s", err.Error())
	}

	return w.GetRegions(area)
}

func listNodeWithRegion(cmd *cobra.Command, args []string) ([]string, error) {
	areaID, err := cmd.Flags().GetString("area-id")
	if len(areaID) == 0 || err != nil {
		return nil, fmt.Errorf("Must set --area-id")
	}

	region, err := cmd.Flags().GetString("region")
	if len(region) == 0 || err != nil {
		return nil, fmt.Errorf("Must set --region")
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("Please specify the name of the config file")
	}

	configFilePath := args[0]
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist.", configFilePath)
	}

	cfg, err := config.ParseConfig(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("parse config error %s", err.Error())
	}

	wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return nil, fmt.Errorf("NewWorker %s", err.Error())
	}

	return w.ListNodesWithRegions(areaID, region)
}

var listRegionsCmd = &cobra.Command{
	Use:     "region",
	Short:   "list regions",
	Example: "region --area=asia /path/to/config",
	Run: func(cmd *cobra.Command, args []string) {
		areaList, err := getRegions(cmd, args)
		if err != nil {
			fmt.Println(err)
			return
		}

		tw := tablewriter.New(
			tablewriter.Col("Index"),
			tablewriter.Col("AreaID"),
			tablewriter.Col("Regions"),
		)

		total := 0
		for i, area := range areaList.List {
			regions, err := json.Marshal(area.Regions)
			if err != nil {
				fmt.Println(err)
				continue
			}

			m := map[string]interface{}{
				"Index":   i,
				"AreaID":  area.AreaID,
				"Regions": string(regions),
			}
			tw.Write(m)
			total++
		}

		tw.Flush(os.Stdout)
		fmt.Printf(color.YellowString("\nTotal: %d ", total))
	},
}
