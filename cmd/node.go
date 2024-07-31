package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
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
				url := fmt.Sprintf("%s/project/%s/%s/trace", node.URL, node.ID, info.ID)
				if strings.Contains(url, "wss://") {
					url = strings.Replace(url, "wss://", "https://", 1)
				} else if strings.Contains(url, "ws://") {
					url = strings.Replace(url, "ws://", "http://", 1)
				}
				fmt.Println(url)
				traceNode(url, "")
			}
			// break
		}
	},
}

func traceNode(url, relay string) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Println("err ", err.Error())
		return
	}
	req.Header.Add("User-Timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	// req.Header.Add("Relay", relay)

	transport := &http.Transport{
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}
	client := &http.Client{
		Transport: transport,
	}

	rsp, err := client.Do(req)
	if err != nil {
		fmt.Println("err ", err.Error())
		return
	}

	body, _ := io.ReadAll(rsp.Body)
	if rsp.StatusCode != 200 {
		fmt.Println("status code %d, body %s, url %s", rsp.StatusCode, string(body), url)
		return
	}

	fmt.Println(delayInfo(rsp.Header))
}

func delayInfo(header http.Header) string {
	requestMap := make(map[string]int64)
	userTimestamp := header.Get("User-timestamp")
	userTimestampInt64, err := strconv.ParseInt(userTimestamp, 10, 64)
	if err != nil {
		fmt.Println("parse user timestamp error %s", err.Error())
		return ""
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

	}
	delayInfo += "==>Client[" + fmt.Sprintf("%d", time.Now().UnixMilli()-userTimestampInt64) + "]"

	return delayInfo

}
