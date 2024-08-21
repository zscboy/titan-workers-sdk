package selector

import (
	"fmt"

	worker "github.com/zscboy/titan-workers-sdk"
)

type WebSelector struct {
	node   *worker.Node
	pInfos []*worker.PorjectInfo
}

func NewWebSelector(currentNode *worker.Node, projectInfos []*worker.PorjectInfo) *WebSelector {
	return &WebSelector{node: currentNode, pInfos: projectInfos}
}

func (ws *WebSelector) GetTunInfos(count int) []*TunInfo {
	for _, projectInfo := range ws.pInfos {
		for _, node := range projectInfo.Nodes {
			if node.ID == ws.node.ID {
				url := fmt.Sprintf("%s/project/%s/%s/tun", node.URL, node.ID, projectInfo.ID)
				return []*TunInfo{&TunInfo{NodeID: node.ID, URL: url}}
			}
		}
	}
	return nil
}
