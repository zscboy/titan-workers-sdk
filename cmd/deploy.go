package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	worker "github.com/zscboy/titan-workers-sdk"
	"github.com/zscboy/titan-workers-sdk/config"
)

func deploy(cmd *cobra.Command, args []string) error {
	areaID, err := cmd.Flags().GetString("area-id")
	if len(areaID) == 0 || err != nil {
		return fmt.Errorf("Must set --area-id")
	}

	region, err := cmd.Flags().GetString("region")
	if len(region) == 0 || err != nil {
		return fmt.Errorf("Must set --region")
	}

	name, err := cmd.Flags().GetString("name")
	if len(name) == 0 || err != nil {
		return fmt.Errorf("Must set --name")
	}

	bundleURL, err := cmd.Flags().GetString("bundle-url")
	if len(bundleURL) == 0 || err != nil {
		return fmt.Errorf("Must set --bundle-url")
	}

	replicas, err := cmd.Flags().GetInt("replicas")
	if replicas == 0 || err != nil {
		return fmt.Errorf("Must set --replicas")
	}

	nodes, err := cmd.Flags().GetString("nodes")
	if err != nil {
		return err
	}

	expiration, err := cmd.Flags().GetString("expiration")
	if err != nil {
		return err
	}

	if len(expiration) == 0 {
		expireTime := time.Now().Add(100 * 24 * time.Hour)
		expiration = expireTime.Format("2006-01-02 15:04:05")
	}

	if len(args) == 0 {
		return fmt.Errorf("Please specify the name of the config file")
	}

	configFilePath := args[0]
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist.", configFilePath)
	}

	cfg, err := config.ParseConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("parse config error " + err.Error())
	}

	// fmt.Println("config ", *cfg)
	// fmt.Println("nodes ", nodes)

	wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return fmt.Errorf("NewWorker ", err.Error())
	}

	base := worker.ProjectBase{Name: name, BundleURL: bundleURL, Replicas: replicas}
	req := &worker.ReqCreateProject{Region: region, ProjectBase: base, NodeIDs: nodes, AreaID: areaID, Expiration: expiration}
	// json.Marshal(req)
	// fmt.Printf("req %#v \n", *req)
	// _ = w
	return w.CreateProject(req)
}
