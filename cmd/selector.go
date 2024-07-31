package main

import (
	"fmt"

	worker "github.com/zscboy/titan-workers-sdk"
	"github.com/zscboy/titan-workers-sdk/config"
)

const maxReconnectCount = 10

type sampleSelector struct {
	ServerURL string
}

func newSampleSelector(serverURL string) (*sampleSelector, error) {
	return &sampleSelector{ServerURL: serverURL}, nil
}

func (selector *sampleSelector) GetNodeURL() (string, error) {
	if len(selector.ServerURL) == 0 {
		panic("no access point exist")
	}
	return selector.ServerURL, nil
}

func (selector *sampleSelector) FindNode(nodeID string) (string, error) {
	return "", fmt.Errorf("not implement")
}

// func (selector *sampleSelector) ReconnectCount() {

// }

type customSelector struct {
	worker         worker.Worker
	pInfos         []*worker.PorjectInfo
	config         *config.Config
	currentUseNode *worker.Node
}

func newCustomSelector(config *config.Config) (*customSelector, error) {
	wConfig := &worker.Config{UserName: config.Server.UserName, Password: config.Server.Password, APIServer: config.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return nil, err
	}

	pInfos, err := loadProjects(w)
	if err != nil {
		return nil, err
	}

	if len(pInfos) == 0 {
		return nil, fmt.Errorf("can not find access points")
	}

	return &customSelector{worker: w, pInfos: pInfos, config: config}, nil
}

func (selector *customSelector) GetNodeURL() (string, error) {
	if len(selector.pInfos) == 0 {
		return "", fmt.Errorf("can not find any project exist")
	}

	for _, pInfo := range selector.pInfos {
		for _, node := range pInfo.Nodes {
			if node.ID == selector.config.Node.ID {
				selector.currentUseNode = node
				url := fmt.Sprintf("%s/project/%s/%s/tun", node.URL, node.ID, pInfo.ID)
				return url, nil
			}
		}
	}

	// get first if not exist special node
	for _, pInfo := range selector.pInfos {
		for _, node := range pInfo.Nodes {
			selector.currentUseNode = node
			url := fmt.Sprintf("%s/project/%s/%s/tun", node.URL, node.ID, pInfo.ID)
			return url, nil
		}
	}

	return "", fmt.Errorf("can not find any project exist")
}

func (selector *customSelector) FindNode(nodeID string) (string, error) {
	for _, pInfo := range selector.pInfos {
		for _, node := range pInfo.Nodes {
			if node.ID == nodeID {
				// copy value
				*selector.currentUseNode = *node
				return fmt.Sprintf("%s/project/%s/%s/tun", node.URL, node.ID, pInfo.ID), nil
			}
		}
	}

	return "", fmt.Errorf("can not find node %s", nodeID)
}

func (selector *customSelector) reloadAccessPoints() error {
	log.Debugf("reloadAccessPoints")
	pInfos, err := loadProjects(selector.worker)
	if err != nil {
		return err
	}
	selector.pInfos = pInfos

	return nil
}

func (selector *customSelector) Count() int {
	count := 0
	for _, info := range selector.pInfos {
		count = count + len(info.Nodes)
	}
	return count
}

func (selector *customSelector) ProjectInfos() []*worker.PorjectInfo {
	return selector.pInfos
}

func (selector *customSelector) CurrentNode() *worker.Node {
	return selector.currentUseNode
}

func loadProjects(w worker.Worker) ([]*worker.PorjectInfo, error) {
	projects, err := w.GetProjects(0, 50)
	if err != nil {
		return nil, err
	}

	projectInfos := make([]*worker.PorjectInfo, 0)
	for _, project := range projects {
		projectInfo, err := w.GetProjectInfo(project.ID)
		if err != nil {
			log.Errorf("GetProjectInfo %s %s", project.AreaID, err.Error())
			continue
		}
		projectInfos = append(projectInfos, projectInfo)
	}

	count := 0
	serviceStatus := 1
	pInfos := make([]*worker.PorjectInfo, 0)

	for _, projectInfo := range projectInfos {
		accessPoints := make([]*worker.Node, 0)
		for _, ap := range projectInfo.Nodes {
			if ap.Status != serviceStatus {
				continue
			}

			if len(ap.URL) == 0 {
				continue
			}

			accessPoints = append(accessPoints, ap)
			count++
		}

		if len(accessPoints) == 0 {
			continue
		}

		info := &worker.PorjectInfo{
			ID:        projectInfo.ID,
			Name:      projectInfo.Name,
			BundleURL: projectInfo.BundleURL,
			AreaID:    projectInfo.AreaID,
			Replicas:  projectInfo.Replicas,
			Nodes:     accessPoints,
		}
		pInfos = append(pInfos, info)

	}

	return pInfos, nil
}
