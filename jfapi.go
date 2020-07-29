package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type ServerInfo struct {
	LocalAddress string `json:"LocalAddress"`
	Name         string `json:"ServerName"`
	Version      string `json:"Version"`
	Os           string `json:"OperatingSystem"`
	Id           string `json:"Id"`
}

type Jellyfin struct {
	server        string
	client        string
	version       string
	device        string
	deviceId      string
	useragent     string
	auth          string
	header        map[string]string
	serverInfo    ServerInfo
	username      string
	password      string
	authenticated bool
	accessToken   string
	userId        string
	httpClient    http.Client
	loginParams   map[string]string
}

func (jf *Jellyfin) init(server, client, version, device, deviceId string) error {
	jf.server = server
	jf.client = client
	jf.version = version
	jf.device = device
	jf.deviceId = deviceId
	jf.useragent = fmt.Sprintf("%s/%s", client, version)
	jf.auth = fmt.Sprintf("MediaBrowser Client=%s, Device=%s, DeviceId=%s, Version=%s", client, device, deviceId, version)
	jf.header = map[string]string{
		"Accept":               "application/json",
		"Content-type":         "application/json; charset=UTF-8",
		"X-Application":        jf.useragent,
		"Accept-Charset":       "UTF-8,*",
		"Accept-Encoding":      "gzip",
		"User-Agent":           jf.useragent,
		"X-Emby-Authorization": jf.auth,
	}
	jf.httpClient = http.Client{
		Timeout: 10 * time.Second,
	}
	infoUrl := fmt.Sprintf("%s/System/Info/Public", server)
	req, _ := http.NewRequest("GET", infoUrl, nil)
	resp, err := jf.httpClient.Do(req)
	if err == nil {
		data, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(data, &jf.serverInfo)
	}
	return nil
}

func (jf *Jellyfin) authenticate(username, password string) (int, error) {
	jf.username = username
	jf.password = password
	jf.loginParams = map[string]string{
		"Username": username,
		"Pw":       password,
	}
	loginParams, _ := json.Marshal(jf.loginParams)
	url := fmt.Sprintf("%s/emby/Users/AuthenticateByName", jf.server)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(loginParams))
	for name, value := range jf.header {
		req.Header.Add(name, value)
	}
	resp, err := jf.httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return resp.StatusCode, err
	}
	defer resp.Body.Close()
	var data io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		data, _ = gzip.NewReader(resp.Body)
	default:
		data = resp.Body
	}
	var respData map[string]interface{}
	json.NewDecoder(data).Decode(&respData)
	jf.accessToken = respData["AccessToken"].(string)
	jf.userId = respData["User"].(map[string]interface{})["Id"].(string)
	jf.auth = fmt.Sprintf("MediaBrowser Client=%s, Device=%s, DeviceId=%s, Version=%s, Token=%s", jf.client, jf.device, jf.deviceId, jf.version, jf.accessToken)
	jf.header["X-Emby-Authorization"] = jf.auth
	jf.authenticated = true
	return resp.StatusCode, nil
}

func (jf *Jellyfin) _getReader(url string, params map[string]string) (io.Reader, int, error) {
	var req *http.Request
	if params != nil {
		jsonParams, _ := json.Marshal(params)
		req, _ = http.NewRequest("GET", url, bytes.NewBuffer(jsonParams))
	} else {
		req, _ = http.NewRequest("GET", url, nil)
	}
	for name, value := range jf.header {
		req.Header.Add(name, value)
	}
	resp, err := jf.httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp.StatusCode == 401 && jf.authenticated {
			jf.authenticated = false
			_, authErr := jf.authenticate(jf.username, jf.password)
			if authErr == nil {
				v1, v2, v3 := jf._getReader(url, params)
				return v1, v2, v3
			}
		}
		return nil, resp.StatusCode, err
	}
	defer resp.Body.Close()
	var data io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		data, _ = gzip.NewReader(resp.Body)
	default:
		data = resp.Body
	}
	//var respData map[string]interface{}
	//json.NewDecoder(data).Decode(&respData)
	return data, resp.StatusCode, nil
}

func (jf *Jellyfin) _post(url string, data map[string]interface{}, response bool) (io.Reader, int, error) {
	params, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(params))
	for name, value := range jf.header {
		req.Header.Add(name, value)
	}
	resp, err := jf.httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp.StatusCode == 401 && jf.authenticated {
			jf.authenticated = false
			_, authErr := jf.authenticate(jf.username, jf.password)
			if authErr == nil {
				v1, v2, v3 := jf._post(url, data, response)
				return v1, v2, v3
			}
		}
		return nil, resp.StatusCode, err
	}
	if response {
		defer resp.Body.Close()
		var outData io.Reader
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			outData, _ = gzip.NewReader(resp.Body)
		default:
			outData = resp.Body
		}
		return outData, resp.StatusCode, nil
	}
	return nil, resp.StatusCode, nil
}

func (jf *Jellyfin) getUsers(public bool) ([]map[string]interface{}, int, error) {
	var result []map[string]interface{}
	var data io.Reader
	var status int
	var err error
	if public {
		url := fmt.Sprintf("%s/emby/Users/Public", jf.server)
		data, status, err = jf._getReader(url, nil)

	} else {
		url := fmt.Sprintf("%s/emby/Users", jf.server)
		data, status, err = jf._getReader(url, jf.loginParams)
	}
	if err != nil || status != 200 {
		return nil, status, err
	}
	json.NewDecoder(data).Decode(&result)
	return result, status, nil

}

func (jf *Jellyfin) userByName(username string, public bool) (map[string]interface{}, int, error) {
	users, status, err := jf.getUsers(public)
	if err != nil || status != 200 {
		return nil, status, err
	}
	for _, user := range users {
		if user["Name"].(string) == username {
			return user, status, nil
		}
	}
	return nil, status, err
}

func (jf *Jellyfin) userById(userId string, public bool) (map[string]interface{}, int, error) {
	if public {
		users, status, err := jf.getUsers(public)
		if err != nil || status != 200 {
			return nil, status, err
		}
		for _, user := range users {
			if user["Id"].(string) == userId {
				return user, status, nil
			}
		}
		return nil, status, err
	} else {
		var result map[string]interface{}
		var data io.Reader
		var status int
		var err error
		url := fmt.Sprintf("%s/emby/Users/%s", jf.server, userId)
		data, status, err = jf._getReader(url, jf.loginParams)
		if err != nil || status != 200 {
			return nil, status, err
		}
		json.NewDecoder(data).Decode(&result)
		return result, status, nil
	}
}

func (jf *Jellyfin) newUser(username, password string) (map[string]interface{}, int, error) {
	url := fmt.Sprintf("%s/emby/Users/New", jf.server)
	stringData := map[string]string{
		"Name":     username,
		"Password": password,
	}
	data := make(map[string]interface{})
	for key, value := range stringData {
		data[key] = value
	}
	reader, status, err := jf._post(url, data, true)
	var recv map[string]interface{}
	json.NewDecoder(reader).Decode(&recv)
	if err != nil || !(status == 200 || status == 204) {
		return nil, status, err
	}
	return recv, status, nil
}

func (jf *Jellyfin) setPolicy(userId string, policy map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Policy", jf.server, userId)
	_, status, err := jf._post(url, policy, false)
	if err != nil || status != 200 {
		return status, err
	}
	return status, nil
}

func (jf *Jellyfin) setConfiguration(userId string, configuration map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Configuration", jf.server, userId)
	_, status, err := jf._post(url, configuration, false)
	return status, err
}

func (jf *Jellyfin) getDisplayPreferences(userId string) (map[string]interface{}, int, error) {
	url := fmt.Sprintf("%s/DisplayPreferences/usersettings?userId=%s&client=emby", jf.server, userId)
	data, status, err := jf._getReader(url, nil)
	if err != nil || !(status == 204 || status == 200) {
		return nil, status, err
	}
	var displayprefs map[string]interface{}
	err = json.NewDecoder(data).Decode(&displayprefs)
	if err != nil {
		return nil, status, err
	}
	return displayprefs, status, nil
}

func (jf *Jellyfin) setDisplayPreferences(userId string, displayprefs map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/DisplayPreferences/usersettings?userId=%s&client=emby", jf.server, userId)
	_, status, err := jf._post(url, displayprefs, false)
	if err != nil || !(status == 204 || status == 200) {
		return status, err
	}
	return status, nil
}
