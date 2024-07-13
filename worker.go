package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Area struct {
	AreaID  string         `json:"area_id"`
	Regions map[string]int `json:"region"`
}

// list":[{"area_id":"Asia-China-Guangdong-Shenzhen","region":["china","hongkong"]}]
type AreaList struct {
	List []*Area
}

type Node struct {
	ID     string `json:"NodeID"`
	URL    string `json:"WsURL"`
	Status int    `json:"status"`
	AreaID string `json:"GeoID"`
	IP     string `json:"IP"`
}

type PorjectInfo struct {
	ID        string  `json:"UUID"`
	Name      string  `json:"Name"`
	BundleURL string  `json:"BundleURL"`
	AreaID    string  `json:"AreaID"`
	Replicas  int     `json:"Replicas"`
	Nodes     []*Node `json:"DetailsList"`
}

type ProjectBase struct {
	Name      string `json:"name"`
	BundleURL string `json:"bundle_url"`
	Replicas  int    `json:"replicas"`
}

type Project struct {
	ID     string `json:"project_id"`
	Status string `json:"status"`
	AreaID string `json:"area_id"`
	Region string `json:"region"`
	ProjectBase
}

type ReqCreateProject struct {
	ProjectBase
	Region     string `json:"region"`
	NodeIDs    string `json:"node_ids"`
	AreaID     string `json:"area_id"`
	Expiration string `json:"expiration"`
}

type ReqUpdatePorjct struct {
	ID string `json:"project_id"`
	ProjectBase
}

type Result struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string
}

type Config struct {
	UserName  string
	Password  string
	APIServer string
}

type Worker interface {
	// DeployProject() error
	UpldateProject(req *ReqUpdatePorjct) error
	CreateProject(req *ReqCreateProject) error
	GetProjects() ([]*Project, error)
	DeleteProject(projectID string) error
	GetProjectInfo(projectID string) (*PorjectInfo, error)
	// area is asia,americas,europe,africa,oceania
	GetRegions(area string) (*AreaList, error)
	ListNodesWithRegions(areaID string, region string) ([]string, error)
}

func NewWorker(cfg *Config) (Worker, error) {
	if len(cfg.UserName) == 0 || len(cfg.Password) == 0 || len(cfg.APIServer) == 0 {
		panic("[UserName][Password][APIServer] can not emtpy")
	}

	w := &worker{config: cfg}

	if err := w.login(); err != nil {
		return nil, err
	}

	return w, nil
}

type worker struct {
	config *Config
	token  string
	expire time.Time
}

func (w *worker) UpldateProject(reqUpdateProject *ReqUpdatePorjct) error {
	buf, err := json.Marshal(reqUpdateProject)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/project/update", w.config.APIServer)
	// Create a new HTTP GET request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	// Add custom headers if needed
	addHeaderToRequest(req, w.token)

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status cod %d, %s", resp.StatusCode, string(buf))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	ret := Result{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return err
	}

	if ret.Code != 0 {
		return fmt.Errorf(string(body))
	}

	return nil
}

func (w *worker) CreateProject(reqCreateProject *ReqCreateProject) error {
	buf, err := json.Marshal(reqCreateProject)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/project/create", w.config.APIServer)
	// Create a new HTTP GET request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	// Add custom headers if needed
	addHeaderToRequest(req, w.token)

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status cod %d, %s", resp.StatusCode, string(buf))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	ret := Result{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return err
	}

	if ret.Code != 0 {
		return fmt.Errorf(string(body))
	}

	return nil
}

func (w *worker) GetProjects() ([]*Project, error) {
	url := fmt.Sprintf("%s/api/v1/project/list", w.config.APIServer)
	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add custom headers if needed
	addHeaderToRequest(req, w.token)

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status cod %d, %s", resp.StatusCode, string(buf))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ret := Result{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(ret.Message)
	}

	// fmt.Println("GetProjects ", string(body))

	dataList := struct {
		List interface{} `json:"list"`
	}{}

	if err = interfaceToStruct(ret.Data, &dataList); err != nil {
		return nil, err
	}

	projects := make([]*Project, 0)
	if err = interfaceToStruct(dataList.List, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func (w *worker) DeleteProject(projectID string) error {
	url := fmt.Sprintf("%s/api/v1/project/delete?project_id=%s", w.config.APIServer, projectID)
	// Create a new HTTP GET request
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	// Add custom headers if needed
	addHeaderToRequest(req, w.token)
	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status cod %d, %s", resp.StatusCode, string(buf))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	ret := Result{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return err
	}

	if ret.Code != 0 {
		return fmt.Errorf(ret.Message)
	}
	return nil
}

func (w *worker) GetProjectInfo(projectID string) (*PorjectInfo, error) {
	url := fmt.Sprintf("%s/api/v1/project/info?project_id=%s", w.config.APIServer, projectID)
	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add custom headers if needed
	addHeaderToRequest(req, w.token)

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status cod %d, %s", resp.StatusCode, string(buf))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// fmt.Println("GetProjectInfo ", string(body))

	ret := Result{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(ret.Message)
	}

	pinfo := PorjectInfo{}
	if err := interfaceToStruct(ret.Data, &pinfo); err != nil {
		return nil, err
	}

	return &pinfo, nil
}

func (w *worker) GetRegions(area string) (*AreaList, error) {
	url := fmt.Sprintf("%s/api/v1/project/regions?region=%s", w.config.APIServer, area)
	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add custom headers if needed
	addHeaderToRequest(req, w.token)

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status code %d, %s", resp.StatusCode, string(buf))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ret := Result{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(ret.Message)
	}

	// fmt.Println("area list ", string(body))
	areaList := &AreaList{}
	if err := interfaceToStruct(ret.Data, &areaList); err != nil {
		return nil, err
	}

	return areaList, nil
}

func (w *worker) ListNodesWithRegions(areaID string, region string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/project/region/nodes?area_id=%s&region=%s", w.config.APIServer, areaID, region)
	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add custom headers if needed
	addHeaderToRequest(req, w.token)

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status code %d, %s", resp.StatusCode, string(buf))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ret := Result{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}

	if ret.Code != 0 {
		return nil, fmt.Errorf(ret.Message)
	}

	// fmt.Println("node list ", string(body))
	// areaList := &AreaList{}
	// if err := interfaceToStruct(ret.Data, &areaList); err != nil {
	// 	return nil, err
	// }

	return nil, nil
}

func (w *worker) login() error {
	loginReq := struct {
		UserName string `json:"username"`
		Password string `json:"password"`
	}{
		UserName: w.config.UserName,
		Password: w.config.Password,
	}

	buf, err := json.Marshal(loginReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/user/login", w.config.APIServer)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	// Add custom headers if needed
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("cache-control", "no-cache")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status code %d, %s", resp.StatusCode, string(buf))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	ret := Result{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return err
	}

	if ret.Code != 0 {
		return fmt.Errorf(ret.Message)
	}

	loginResult := struct {
		Token  string `json:"token"`
		Expire string `json:"expire"`
	}{}

	if err := interfaceToStruct(ret.Data, &loginResult); err != nil {
		return err
	}

	// fmt.Println("loginResult %#v", loginResult)

	timeFormat := "2006-01-02T15:04:05-07:00"
	expireTime, err := time.Parse(timeFormat, loginResult.Expire)
	if err != nil {
		return err
	}

	w.expire = expireTime
	w.token = loginResult.Token
	return nil
}

func interfaceToStruct(input interface{}, output interface{}) error {
	jsonData, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("error marshaling map to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonData, output); err != nil {
		return fmt.Errorf("error unmarshaling JSON to struct: %w", err)
	}

	return nil
}

func addHeaderToRequest(req *http.Request, token string) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("jwtAuthorization", "Bearer "+token)
}
