package jfapi

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hrfee/jfa-go/common"
)

type serverInfo struct {
	LocalAddress string `json:"LocalAddress"`
	Name         string `json:"ServerName"`
	Version      string `json:"Version"`
	OS           string `json:"OperatingSystem"`
	ID           string `json:"Id"`
}

// Jellyfin represents a running Jellyfin instance.
type Jellyfin struct {
	Server         string
	client         string
	version        string
	device         string
	deviceID       string
	useragent      string
	auth           string
	header         map[string]string
	ServerInfo     serverInfo
	Username       string
	password       string
	Authenticated  bool
	AccessToken    string
	userID         string
	httpClient     *http.Client
	loginParams    map[string]string
	userCache      []map[string]interface{}
	CacheExpiry    time.Time
	cacheLength    int
	noFail         bool
	Hyphens        bool
	timeoutHandler common.TimeoutHandler
}

// NewJellyfin returns a new Jellyfin object.
func NewJellyfin(server, client, version, device, deviceID string, timeoutHandler common.TimeoutHandler, cacheTimeout int) (*Jellyfin, error) {
	jf := &Jellyfin{}
	jf.Server = server
	jf.client = client
	jf.version = version
	jf.device = device
	jf.deviceID = deviceID
	jf.useragent = fmt.Sprintf("%s/%s", client, version)
	jf.timeoutHandler = timeoutHandler
	jf.auth = fmt.Sprintf("MediaBrowser Client=\"%s\", Device=\"%s\", DeviceId=\"%s\", Version=\"%s\"", client, device, deviceID, version)
	jf.header = map[string]string{
		"Accept":               "application/json",
		"Content-type":         "application/json; charset=UTF-8",
		"X-Application":        jf.useragent,
		"Accept-Charset":       "UTF-8,*",
		"Accept-Encoding":      "gzip",
		"User-Agent":           jf.useragent,
		"X-Emby-Authorization": jf.auth,
	}
	jf.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	infoURL := fmt.Sprintf("%s/System/Info/Public", server)
	req, _ := http.NewRequest("GET", infoURL, nil)
	resp, err := jf.httpClient.Do(req)
	defer jf.timeoutHandler()
	if err == nil {
		data, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(data, &jf.ServerInfo)
	}
	jf.cacheLength = cacheTimeout
	jf.CacheExpiry = time.Now()
	return jf, nil
}

// Authenticate attempts to authenticate using a username & password
func (jf *Jellyfin) Authenticate(username, password string) (map[string]interface{}, int, error) {
	jf.Username = username
	jf.password = password
	jf.loginParams = map[string]string{
		"Username": username,
		"Pw":       password,
		"Password": password,
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(jf.loginParams)
	if err != nil {
		return nil, 0, err
	}
	// loginParams, _ := json.Marshal(jf.loginParams)
	url := fmt.Sprintf("%s/Users/authenticatebyname", jf.Server)
	req, err := http.NewRequest("POST", url, buffer)
	defer jf.timeoutHandler()
	if err != nil {
		return nil, 0, err
	}
	for name, value := range jf.header {
		req.Header.Add(name, value)
	}
	resp, err := jf.httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
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
	var respData map[string]interface{}
	json.NewDecoder(data).Decode(&respData)
	jf.AccessToken = respData["AccessToken"].(string)
	user := respData["User"].(map[string]interface{})
	jf.userID = respData["User"].(map[string]interface{})["Id"].(string)
	jf.auth = fmt.Sprintf("MediaBrowser Client=\"%s\", Device=\"%s\", DeviceId=\"%s\", Version=\"%s\", Token=\"%s\"", jf.client, jf.device, jf.deviceID, jf.version, jf.AccessToken)
	jf.header["X-Emby-Authorization"] = jf.auth
	jf.Authenticated = true
	return user, resp.StatusCode, nil
}

func (jf *Jellyfin) get(url string, params map[string]string) (string, int, error) {
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
	defer jf.timeoutHandler()
	if err != nil || resp.StatusCode != 200 {
		if resp.StatusCode == 401 && jf.Authenticated {
			jf.Authenticated = false
			_, _, authErr := jf.Authenticate(jf.Username, jf.password)
			if authErr == nil {
				v1, v2, v3 := jf.get(url, params)
				return v1, v2, v3
			}
		}
		return "", resp.StatusCode, err
	}
	defer resp.Body.Close()
	var data io.Reader
	encoding := resp.Header.Get("Content-Encoding")
	switch encoding {
	case "gzip":
		data, _ = gzip.NewReader(resp.Body)
	default:
		data = resp.Body
	}
	buf := new(strings.Builder)
	io.Copy(buf, data)
	//var respData map[string]interface{}
	//json.NewDecoder(data).Decode(&respData)
	return buf.String(), resp.StatusCode, nil
}

func (jf *Jellyfin) post(url string, data map[string]interface{}, response bool) (string, int, error) {
	params, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(params))
	for name, value := range jf.header {
		req.Header.Add(name, value)
	}
	resp, err := jf.httpClient.Do(req)
	defer jf.timeoutHandler()
	if err != nil || resp.StatusCode != 200 {
		if resp.StatusCode == 401 && jf.Authenticated {
			jf.Authenticated = false
			_, _, authErr := jf.Authenticate(jf.Username, jf.password)
			if authErr == nil {
				v1, v2, v3 := jf.post(url, data, response)
				return v1, v2, v3
			}
		}
		return "", resp.StatusCode, err
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
		buf := new(strings.Builder)
		io.Copy(buf, outData)
		return buf.String(), resp.StatusCode, nil
	}
	return "", resp.StatusCode, nil
}

// DeleteUser deletes the user corresponding to the provided ID.
func (jf *Jellyfin) DeleteUser(userID string) (int, error) {
	url := fmt.Sprintf("%s/Users/%s", jf.Server, userID)
	req, _ := http.NewRequest("DELETE", url, nil)
	for name, value := range jf.header {
		req.Header.Add(name, value)
	}
	resp, err := jf.httpClient.Do(req)
	defer jf.timeoutHandler()
	return resp.StatusCode, err
}

// GetUsers returns all (visible) users on the Jellyfin instance.
func (jf *Jellyfin) GetUsers(public bool) ([]map[string]interface{}, int, error) {
	var result []map[string]interface{}
	var data string
	var status int
	var err error
	if time.Now().After(jf.CacheExpiry) {
		if public {
			url := fmt.Sprintf("%s/users/public", jf.Server)
			data, status, err = jf.get(url, nil)
		} else {
			url := fmt.Sprintf("%s/users", jf.Server)
			data, status, err = jf.get(url, jf.loginParams)
		}
		if err != nil || status != 200 {
			return nil, status, err
		}
		json.Unmarshal([]byte(data), &result)
		jf.userCache = result
		jf.CacheExpiry = time.Now().Add(time.Minute * time.Duration(jf.cacheLength))
		if id, ok := result[0]["Id"]; ok {
			if id.(string)[8] == '-' {
				jf.Hyphens = true
			}
		}
		return result, status, nil
	}
	return jf.userCache, 200, nil
}

// UserByName returns the user corresponding to the provided username.
func (jf *Jellyfin) UserByName(username string, public bool) (map[string]interface{}, int, error) {
	var match map[string]interface{}
	find := func() (map[string]interface{}, int, error) {
		users, status, err := jf.GetUsers(public)
		if err != nil || status != 200 {
			return nil, status, err
		}
		for _, user := range users {
			if user["Name"].(string) == username {
				return user, status, err
			}
		}
		return nil, status, err
	}
	match, status, err := find()
	if match == nil {
		jf.CacheExpiry = time.Now()
		match, status, err = find()
	}
	return match, status, err
}

// UserByID returns the user corresponding to the provided ID.
func (jf *Jellyfin) UserByID(userID string, public bool) (map[string]interface{}, int, error) {
	if jf.CacheExpiry.After(time.Now()) {
		for _, user := range jf.userCache {
			if user["Id"].(string) == userID {
				return user, 200, nil
			}
		}
	}
	if public {
		users, status, err := jf.GetUsers(public)
		if err != nil || status != 200 {
			return nil, status, err
		}
		for _, user := range users {
			if user["Id"].(string) == userID {
				return user, status, nil
			}
		}
		return nil, status, err
	}
	var result map[string]interface{}
	var data string
	var status int
	var err error
	url := fmt.Sprintf("%s/users/%s", jf.Server, userID)
	data, status, err = jf.get(url, jf.loginParams)
	if err != nil || status != 200 {
		return nil, status, err
	}
	json.Unmarshal([]byte(data), &result)
	return result, status, nil
}

// NewUser creates a new user with the provided username and password.
func (jf *Jellyfin) NewUser(username, password string) (map[string]interface{}, int, error) {
	url := fmt.Sprintf("%s/Users/New", jf.Server)
	stringData := map[string]string{
		"Name":     username,
		"Password": password,
	}
	data := make(map[string]interface{})
	for key, value := range stringData {
		data[key] = value
	}
	response, status, err := jf.post(url, data, true)
	var recv map[string]interface{}
	json.Unmarshal([]byte(response), &recv)
	if err != nil || !(status == 200 || status == 204) {
		return nil, status, err
	}
	return recv, status, nil
}

// SetPolicy sets the access policy for the user corresponding to the provided ID.
func (jf *Jellyfin) SetPolicy(userID string, policy map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Policy", jf.Server, userID)
	_, status, err := jf.post(url, policy, false)
	if err != nil || status != 200 {
		return status, err
	}
	return status, nil
}

// SetConfiguration sets the configuration (part of homescreen layout) for the user corresponding to the provided ID.
func (jf *Jellyfin) SetConfiguration(userID string, configuration map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Configuration", jf.Server, userID)
	_, status, err := jf.post(url, configuration, false)
	return status, err
}

// GetDisplayPreferences gets the displayPreferences (part of homescreen layout) for the user corresponding to the provided ID.
func (jf *Jellyfin) GetDisplayPreferences(userID string) (map[string]interface{}, int, error) {
	url := fmt.Sprintf("%s/DisplayPreferences/usersettings?userId=%s&client=emby", jf.Server, userID)
	data, status, err := jf.get(url, nil)
	if err != nil || !(status == 204 || status == 200) {
		return nil, status, err
	}
	var displayprefs map[string]interface{}
	err = json.Unmarshal([]byte(data), &displayprefs)
	if err != nil {
		return nil, status, err
	}
	return displayprefs, status, nil
}

// SetDisplayPreferences sets the displayPreferences (part of homescreen layout) for the user corresponding to the provided ID.
func (jf *Jellyfin) SetDisplayPreferences(userID string, displayprefs map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/DisplayPreferences/usersettings?userId=%s&client=emby", jf.Server, userID)
	_, status, err := jf.post(url, displayprefs, false)
	if err != nil || !(status == 204 || status == 200) {
		return status, err
	}
	return status, nil
}
