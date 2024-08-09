package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	worker "github.com/zscboy/titan-workers-sdk"
	"github.com/zscboy/titan-workers-sdk/config"
	"github.com/zscboy/titan-workers-sdk/tablewriter"
)

func listNodes(cmd *cobra.Command, args []string) ([]*worker.PorjectInfo, error) {
	page, err := cmd.Flags().GetInt("page")
	if err != nil {
		return nil, fmt.Errorf("Must set --page")
	}

	size, err := cmd.Flags().GetInt("size")
	if size == 0 || err != nil {
		return nil, fmt.Errorf("Must set --size")
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
		return nil, fmt.Errorf("parse config error " + err.Error())
	}

	wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return nil, fmt.Errorf("NewWorker %s", err.Error())
	}

	projects, err := w.GetProjects(page, size)
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
			tablewriter.Col("url"),
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
					"url":    node.URL,
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

var checkDelayCmd = &cobra.Command{
	Use:     "delay-check",
	Short:   "check nodes delay",
	Example: "delay-check /path/to/config",
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

		if len(args) == 0 {
			fmt.Println("Please specify the name of the config file")
			return
		}

		configFilePath := args[0]
		if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
			fmt.Printf("%s does not exist.\n", configFilePath)
			return
		}

		cfg, err := config.ParseConfig(configFilePath)
		if err != nil {
			fmt.Errorf("parse config error " + err.Error())
			return
		}

		wConfig := &worker.Config{UserName: cfg.Server.UserName, Password: cfg.Server.Password, APIServer: cfg.Server.URL}
		w, err := worker.NewWorker(wConfig)
		if err != nil {
			return
		}

		delayInfos := make([]*DelayInfo, 0)
		for _, info := range pInfos {

			tuns, err := w.GetTunnels(info.ID)
			if err != nil {
				continue
			}

			if len(tuns) == 0 {
				continue
			}
			for _, node := range info.Nodes {
				if node.Status != 1 {
					continue
				}
				if len(node.URL) == 0 {
					continue
				}
				url := fmt.Sprintf("%s/project/%s/%s/tun", node.URL, node.ID, info.ID)
				delyaInfo, err := traceNode(url, node.URL)
				if err != nil {
					fmt.Println(err)
					continue
				}
				delyaInfo.AreaID = node.AreaID

				delayInfos = append(delayInfos, delyaInfo)
			}

		}

		// sort by delay time
		sort.Slice(delayInfos, func(i, j int) bool {
			return delayInfos[i].Delay < delayInfos[j].Delay
		})

		for _, delayInfo := range delayInfos {
			fmt.Println(delayInfo.Info, delayInfo.AreaID)
		}
	},
}

func traceNode(url, relay string) (*DelayInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	header := make(http.Header)
	header.Set("User-Timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	// header.Add("Relay", relay)

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, url, header)
	if err != nil {
		if resp != nil {
			buf, _ := io.ReadAll(resp.Body)
			fmt.Println("dial err ", err.Error(), string(buf), url)
		} else {
			fmt.Println("dial err ", err.Error(), url)
		}
		return nil, err
	}
	defer conn.Close()

	// fmt.Println("header ", resp.Header)
	return delayInfo(resp.Header)
}

type DelayInfo struct {
	NodeID string
	Delay  int64
	Info   string
	AreaID string
}

func delayInfo(header http.Header) (*DelayInfo, error) {
	requestMap := make(map[string]int64)
	userTimestamp := header.Get("User-timestamp")
	userTimestampInt64, err := strconv.ParseInt(userTimestamp, 10, 64)
	if err != nil {
		fmt.Println("parse user timestamp error %s ", err.Error())
		return nil, err
	}

	nodes := strings.Split(header.Get("Request-Nodes"), ",")
	requestNodesTimestamps := header.Get("Request-Nodes-Timestamps")

	for i, timestamp := range strings.Split(requestNodesTimestamps, ",") {
		nodeTimestamp, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
		if err != nil {
			fmt.Println("parse node timestamp error %s", err.Error())
			continue
		}
		requestMap[strings.TrimSpace(nodes[i])] = nodeTimestamp
	}

	delayInfo := ""

	edgeNodeID := ""
	nodes = header.Values("Response-Nodes")
	responseNodesTimestamps := header.Values("Response-Nodes-Timestamps")

	for i, timestamp := range responseNodesTimestamps {
		nodeTimestamp, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
		if err != nil {
			fmt.Println("parse node timestamp error %s", err.Error())
			continue
		}
		nodeID := nodes[i]
		delayInfo += "==>" + nodeID + "[" + fmt.Sprintf("%d", nodeTimestamp-requestMap[nodeID]) + "]"

		if strings.Contains("e_", nodeID) {
			edgeNodeID = nodeID
		}

	}
	delay := time.Now().UnixMilli() - userTimestampInt64
	delayInfo += "==>Client[" + fmt.Sprintf("%d", delay) + "]"

	return &DelayInfo{NodeID: edgeNodeID, Delay: delay, Info: delayInfo}, nil

}
