package web

import (
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	worker "github.com/zscboy/titan-workers-sdk"
)

//go:embed static/index.html
var indexHTML string

//go:embed static/*
var StaticFiles embed.FS

var log = logging.Logger("web")

type Web struct {
	projectInfos []*worker.PorjectInfo
	currentNode  *worker.Node
	areas        Areas
}

type Countrys map[string][]*worker.Node
type Areas map[string]Countrys

func getAreaFromAreaID(areaID string) string {
	values := strings.Split(areaID, "-")
	return values[0]
}

func getCountryFromAreaID(areaID string) string {
	values := strings.Split(areaID, "-")
	if len(values) > 1 {
		return values[1]
	}
	return ""
}

func NewWeb(pInfos []*worker.PorjectInfo, currentNode *worker.Node) *Web {
	w := &Web{currentNode: currentNode}

	areas := make(Areas)
	for _, pInfo := range pInfos {
		for _, node := range pInfo.Nodes {
			if node.Status != 1 {
				continue
			}

			area := getAreaFromAreaID(node.AreaID)
			if len(area) == 0 {
				log.Infof("nodeID %s areaID %s", node.ID, node.AreaID)
				continue
			}
			country := getCountryFromAreaID(node.AreaID)
			if len(country) == 0 {
				log.Infof("nodeID %s areaID %s", node.ID, node.AreaID)
				continue
			}

			a, ok := areas[area]
			if !ok {
				a = make(Countrys)
			}

			_, ok = a[country]
			if !ok {
				a[country] = make([]*worker.Node, 0)
			}

			a[country] = append(a[country], node)
			areas[area] = a
		}
	}

	w.areas = areas
	return w
}

type Option struct {
	Value string
	Text  string
}

type TemplateData struct {
	Options []Option
	Node    worker.Node
}

func (web *Web) WebHandler(w http.ResponseWriter, r *http.Request) {
	areas := make([]Option, 0, len(web.areas))
	for area := range web.areas {
		areas = append(areas, Option{Value: area, Text: area})
	}

	temp := TemplateData{Node: *web.currentNode, Options: areas}

	t, err := template.New("index").Parse(indexHTML)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, temp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (web *Web) GetCountryOptions(w http.ResponseWriter, r *http.Request) {
	areaOption := r.URL.Query().Get("areaOption")
	countrys := web.areas[areaOption]

	countryOpts := make([]Option, 0, len(countrys))
	for k := range countrys {
		countryOpts = append(countryOpts, Option{Value: fmt.Sprintf("%s-%s", areaOption, k), Text: k})
	}
	log.Infof("countryOpts %#v", countryOpts)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(countryOpts)
}

func (web *Web) GetNodeOptions(w http.ResponseWriter, r *http.Request) {
	countryOption := r.URL.Query().Get("countryOption")
	var area = ""
	var country = ""
	values := strings.Split(countryOption, "-")
	if len(values) > 1 {
		area = values[0]
		country = values[1]
	}

	nodes := web.areas[area][country]

	nodeOpts := make([]Option, 0, len(nodes))
	for _, node := range nodes {
		nodeOpts = append(nodeOpts, Option{Value: node.ID, Text: node.ID})
	}
	// log.Infof("nodeOpts %s %#v", countryOption, nodeOpts)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeOpts)
}

func (web *Web) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		nodeID := r.FormValue("nodeSelect")
		if len(nodeID) == 0 {
			fmt.Fprintf(w, "You must select a node")
			return
		}

		url := fmt.Sprintf("http://localhost:1082/change?id=%s", nodeID)
		rsp, err := http.Post(url, "application/json", nil)
		if err != nil {
			fmt.Fprintf(w, "Set node failed: %s", err.Error())
			return
		}

		if rsp.StatusCode != http.StatusOK {
			buf, _ := io.ReadAll(rsp.Body)
			fmt.Fprintf(w, "Set node failed, status code %d, error %s", rsp.StatusCode, string(buf))
			return
		}

		// refresh web
	}
}
