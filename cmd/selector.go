package main

import (
	"fmt"

	worker "github.com/zscboy/titan-workers-sdk"
	"github.com/zscboy/titan-workers-sdk/config"
	"github.com/zscboy/titan-workers-sdk/proxy"
)

const maxReconnectCount = 10

type sampleSelector struct {
	ServerURL string
}

func newSampleSelector(serverURL string) (proxy.Selector, error) {
	return &sampleSelector{ServerURL: serverURL}, nil
}

func (selector *sampleSelector) GetServerURL() (string, error) {
	if len(selector.ServerURL) == 0 {
		panic("no access point exist")
	}
	return selector.ServerURL, nil
}

func (selector *sampleSelector) ReconnectCount() {

}

type customSelector struct {
	worker         worker.Worker
	urls           []string
	nextIdx        int
	reconnectCount int
	config         *config.Config
}

func newCustomSelector(config *config.Config) (proxy.Selector, error) {
	wConfig := &worker.Config{UserName: config.Server.UserName, Password: config.Server.Password, APIServer: config.Server.URL}
	w, err := worker.NewWorker(wConfig)
	if err != nil {
		return nil, err
	}

	urls, err := loadAccessPoints(w, config)
	if err != nil {
		return nil, err
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("can not find access points")
	}

	log.Infof("get access point %d", len(urls))
	return &customSelector{worker: w, urls: urls, nextIdx: 0, reconnectCount: 0, config: config}, nil
}

func (selector *customSelector) GetServerURL() (string, error) {
	if selector.reconnectCount > maxReconnectCount {
		selector.reloadAccessPoints()
	}

	if len(selector.urls) == 0 {
		return "", fmt.Errorf("can not find access points")
	}

	idx := selector.nextIdx % len(selector.urls)
	url := selector.urls[idx]

	selector.nextIdx = idx + 1
	if selector.nextIdx == len(selector.urls) {
		selector.nextIdx = 0
	}
	return url, nil
}

func (selector *customSelector) ReconnectCount() {
	selector.reconnectCount++
}

func (selector *customSelector) reloadAccessPoints() error {
	log.Debugf("reloadAccessPoints")
	urls, err := loadAccessPoints(selector.worker, selector.config)
	if err != nil {
		return err
	}
	selector.urls = urls

	selector.reconnectCount = 0
	selector.nextIdx = 0

	return nil
}

func loadAccessPoints(w worker.Worker, config *config.Config) ([]string, error) {
	regionMap := make(map[string]struct{})
	for _, project := range config.Projects {
		regionMap[project.Region] = struct{}{}
	}

	projects, err := w.GetProjects()
	if err != nil {
		return nil, err
	}

	projectInfos := make([]*worker.PorjectInfo, 0, len(config.Projects))
	for _, project := range projects {
		if _, ok := regionMap[project.Region]; ok {
			projectInfo, err := w.GetProjectInfo(project.ID)
			if err != nil {
				log.Errorf("GetProjectInfo %s %s", project.AreaID, err.Error())
				continue
			}
			projectInfos = append(projectInfos, projectInfo)
		}
	}

	urls := make([]string, 0)
	for _, projectInfo := range projectInfos {
		for _, ap := range projectInfo.AccessPoints {
			if len(ap.URL) != 0 {
				url := fmt.Sprintf("%s/project/%s/%s/tun", ap.URL, ap.L2NodeID, projectInfo.ID)
				urls = append(urls, url)
			}
		}

	}

	return urls, nil
}
