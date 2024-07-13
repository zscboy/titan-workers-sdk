package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	worker "github.com/zscboy/titan-workers-sdk"
	"github.com/zscboy/titan-workers-sdk/config"
	"github.com/zscboy/titan-workers-sdk/tablewriter"
)

func listNodes(cmd *cobra.Command, args []string) ([]*worker.PorjectInfo, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("Please specify the name of the config file")
	}

	configFilePath := args[0]
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist.", configFilePath)
	}

	cfg, err := config.ParseConfig(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("parse config error " + err.Error())
	}

	wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return nil, fmt.Errorf("NewWorker %s", err.Error())
	}

	projects, err := w.GetProjects()
	if err != nil {
		return nil, err
	}

	projectInfos := make([]*worker.PorjectInfo, 0)
	for _, project := range projects {
		projectInfo, err := w.GetProjectInfo(project.ID)
		if err != nil {
			// log.Errorf("GetProjectInfo %s %s", project.AreaID, err.Error())
			continue
		}
		projectInfos = append(projectInfos, projectInfo)
	}
	return projectInfos, nil
}

var listNodesCmd = &cobra.Command{
	Use:     "node",
	Short:   "list all nodes",
	Example: "node /path/to/config",
	Run: func(cmd *cobra.Command, args []string) {
		pInfos, err := listNodes(cmd, args)
		if err != nil {
			fmt.Println("list projects ", err.Error())
			return
		}

		if len(pInfos) == 0 {
			fmt.Println("no node exist")
			return
		}

		tw := tablewriter.New(
			tablewriter.Col("Num"),
			tablewriter.Col("NodeID"),
			tablewriter.Col("AreaID"),
			tablewriter.Col("IP"),
			tablewriter.Col("Status"),
		)

		total := 0
		for _, info := range pInfos {
			for _, node := range info.Nodes {
				m := map[string]interface{}{
					"Num":    total,
					"NodeID": node.ID,
					"AreaID": node.AreaID,
					"IP":     node.IP,
					"Status": statusToString(node.Status),
				}
				tw.Write(m)
				total++
			}
		}

		tw.Flush(os.Stdout)
		fmt.Printf(color.YellowString("\nTotal: %d ", total))

	},
}

var setNodeCmd = &cobra.Command{
	Use:     "set",
	Short:   "set current use node",
	Example: "set node-id",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Must set node-id")
			return
		}

		nodeID := args[0]
		url := fmt.Sprintf("http://localhost:1082/change?id=%s", nodeID)

		rsp, err := http.Post(url, "application/json", nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		if rsp.StatusCode != http.StatusOK {
			buf, _ := io.ReadAll(rsp.Body)
			fmt.Printf("status code %d, error: %s\n", rsp.StatusCode, string(buf))
			return
		}

		url = fmt.Sprintf("http://localhost:1082/query")
		rsp, err = http.Get(url)
		if err != nil {
			fmt.Println(err)
			return
		}

		buf, _ := io.ReadAll(rsp.Body)
		data := struct {
			NodeID string `json:"NodeID"`
			WsURL  string `json:"WsURL"`
			Status int    `json:"status"`
			AreaID string `json:"GeoID"`
			IP     string `json:"IP"`
		}{}

		err = json.Unmarshal(buf, &data)
		if err != nil {
			fmt.Println(err)
			return
		}

		tw := tablewriter.New(
			tablewriter.Col("NodeID"),
			tablewriter.Col("AreaID"),
			tablewriter.Col("IP"),
			tablewriter.Col("Status"),
		)

		m := map[string]interface{}{
			"NodeID": data.NodeID,
			"AreaID": data.AreaID,
			"IP":     data.IP,
			"Status": statusToString(data.Status),
		}
		tw.Write(m)

		tw.Flush(os.Stdout)

	},
}

var queryNodeCmd = &cobra.Command{
	Use:     "query",
	Short:   "query current use node",
	Example: "query",
	Run: func(cmd *cobra.Command, args []string) {
		url := fmt.Sprintf("http://localhost:1082/query")
		rsp, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
			return
		}

		buf, _ := io.ReadAll(rsp.Body)
		data := struct {
			NodeID string `json:"NodeID"`
			WsURL  string `json:"WsURL"`
			Status int    `json:"status"`
			AreaID string `json:"GeoID"`
			IP     string `json:"IP"`
		}{}

		err = json.Unmarshal(buf, &data)
		if err != nil {
			fmt.Println(err)
			return
		}

		tw := tablewriter.New(
			tablewriter.Col("NodeID"),
			tablewriter.Col("AreaID"),
			tablewriter.Col("IP"),
			tablewriter.Col("Status"),
		)

		m := map[string]interface{}{
			"NodeID": data.NodeID,
			"AreaID": data.AreaID,
			"IP":     data.IP,
			"Status": statusToString(data.Status),
		}
		tw.Write(m)

		tw.Flush(os.Stdout)
	},
}

func statusToString(status int) string {
	if status == 0 {
		return "starting"
	} else if status == 1 {
		return "started"
	} else if status == 2 {
		return "failed"
	} else if status == 3 {
		return "offline"
	}

	return "unknow"
}
