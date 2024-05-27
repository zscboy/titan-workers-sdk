package proxy

import (
	"fmt"

	worker "github.com/zscboy/titan-workers-sdk"
)

type AccessPoint interface {
	GetServerURL() (string, error)
	RefreshServerURL()
}

func NewWorkerAccessPoint(projectID string, w worker.Worker) AccessPoint {
	return &accessPoint{ProjectID: projectID, worker: w}
}

type accessPoint struct {
	ProjectID string
	ServerURL string
	L2NodeID  string
	worker    worker.Worker
}

func (ap *accessPoint) GetServerURL() (string, error) {
	if len(ap.ServerURL) != 0 {
		return ap.ServerURL, nil
	}

	info, err := ap.worker.GetProjectInfo(ap.ProjectID)
	if err != nil {
		return "", err
	}

	if len(info.AccessPoints) == 0 {
		return "", fmt.Errorf("no access point exist for project %s", ap.ProjectID)
	}

	accssPoint := info.AccessPoints[0]
	if len(accssPoint.URL) == 0 || len(accssPoint.L2NodeID) == 0 {
		return "", fmt.Errorf("can not get project %s access point", ap.ProjectID)
	}

	url := fmt.Sprintf("%s/project/%s/%s", accssPoint.URL, accssPoint.L2NodeID, ap.ProjectID)

	ap.ServerURL = url
	ap.L2NodeID = accssPoint.L2NodeID

	return url, nil
}

func (ap *accessPoint) RefreshServerURL() {
	info, err := ap.worker.GetProjectInfo(ap.ProjectID)
	if err != nil {
		log.Errorf("Get project %s info failed %s", ap.ProjectID, err.Error())
		return
	}

	if len(info.AccessPoints) == 0 {
		log.Errorf("no access point exist for project %s", ap.ProjectID)
		return
	}

	isUpdateURL := false
	for _, accessPoint := range info.AccessPoints {
		if len(accessPoint.URL) != 0 && len(accessPoint.L2NodeID) != 0 &&
			accessPoint.L2NodeID == ap.L2NodeID {
			ap.ServerURL = fmt.Sprintf("%s/project/%s/%s", accessPoint.URL, accessPoint.L2NodeID, ap.ProjectID)
			isUpdateURL = true
			break
		}
	}

	if !isUpdateURL {
		log.Errorf("can not update access point url")
	}
}
