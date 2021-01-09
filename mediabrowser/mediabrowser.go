package mediabrowser

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

type serverType bool

var JellyfinServer serverType = false
var EmbyServer serverType = true

type serverInfo struct {
	LocalAddress string `json:"LocalAddress"`
	Name         string `json:"ServerName"`
	Version      string `json:"Version"`
	OS           string `json:"OperatingSystem"`
	ID           string `json:"Id"`
}

// MediaBrowser is an api instance of Jellyfin/Emby.
type MediaBrowser struct {
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
	serverType     serverType
	timeoutHandler common.TimeoutHandler
}

// NewServer returns a new Jellyfin object.
func NewServer(st serverType, server, client, version, device, deviceID string, timeoutHandler common.TimeoutHandler, cacheTimeout int) (*MediaBrowser, error) {
	mb := &MediaBrowser{}
	mb.serverType = st
	mb.Server = server
	mb.client = client
	mb.version = version
	mb.device = device
	mb.deviceID = deviceID
	mb.useragent = fmt.Sprintf("%s/%s", client, version)
	mb.timeoutHandler = timeoutHandler
	mb.auth = fmt.Sprintf("MediaBrowser Client=\"%s\", Device=\"%s\", DeviceId=\"%s\", Version=\"%s\"", client, device, deviceID, version)
	mb.header = map[string]string{
		"Accept":               "application/json",
		"Content-type":         "application/json; charset=UTF-8",
		"X-Application":        mb.useragent,
		"Accept-Charset":       "UTF-8,*",
		"Accept-Encoding":      "gzip",
		"User-Agent":           mb.useragent,
		"X-Emby-Authorization": mb.auth,
	}
	mb.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	infoURL := fmt.Sprintf("%s/System/Info/Public", server)
	req, _ := http.NewRequest("GET", infoURL, nil)
	resp, err := mb.httpClient.Do(req)
	defer mb.timeoutHandler()
	if err == nil {
		data, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(data, &mb.ServerInfo)
	}
	mb.cacheLength = cacheTimeout
	mb.CacheExpiry = time.Now()
	return mb, nil
}

func (mb *MediaBrowser) get(url string, params map[string]string) (string, int, error) {
	var req *http.Request
	if params != nil {
		jsonParams, _ := json.Marshal(params)
		req, _ = http.NewRequest("GET", url, bytes.NewBuffer(jsonParams))
	} else {
		req, _ = http.NewRequest("GET", url, nil)
	}
	for name, value := range mb.header {
		req.Header.Add(name, value)
	}
	resp, err := mb.httpClient.Do(req)
	defer mb.timeoutHandler()
	if err != nil || resp.StatusCode != 200 {
		if resp.StatusCode == 401 && mb.Authenticated {
			mb.Authenticated = false
			_, _, authErr := mb.Authenticate(mb.Username, mb.password)
			if authErr == nil {
				v1, v2, v3 := mb.get(url, params)
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

func (mb *MediaBrowser) post(url string, data map[string]interface{}, response bool) (string, int, error) {
	params, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(params))
	for name, value := range mb.header {
		req.Header.Add(name, value)
	}
	resp, err := mb.httpClient.Do(req)
	defer mb.timeoutHandler()
	if err != nil || resp.StatusCode != 200 {
		if resp.StatusCode == 401 && mb.Authenticated {
			mb.Authenticated = false
			_, _, authErr := mb.Authenticate(mb.Username, mb.password)
			if authErr == nil {
				v1, v2, v3 := mb.post(url, data, response)
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

// Authenticate attempts to authenticate using a username & password
func (mb *MediaBrowser) Authenticate(username, password string) (map[string]interface{}, int, error) {
	mb.Username = username
	mb.password = password
	mb.loginParams = map[string]string{
		"Username": username,
		"Pw":       password,
		"Password": password,
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(mb.loginParams)
	if err != nil {
		return nil, 0, err
	}
	// loginParams, _ := json.Marshal(jf.loginParams)
	url := fmt.Sprintf("%s/Users/authenticatebyname", mb.Server)
	req, err := http.NewRequest("POST", url, buffer)
	defer mb.timeoutHandler()
	if err != nil {
		return nil, 0, err
	}
	for name, value := range mb.header {
		req.Header.Add(name, value)
	}
	resp, err := mb.httpClient.Do(req)
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
	mb.AccessToken = respData["AccessToken"].(string)
	user := respData["User"].(map[string]interface{})
	mb.userID = respData["User"].(map[string]interface{})["Id"].(string)
	mb.auth = fmt.Sprintf("MediaBrowser Client=\"%s\", Device=\"%s\", DeviceId=\"%s\", Version=\"%s\", Token=\"%s\"", mb.client, mb.device, mb.deviceID, mb.version, mb.AccessToken)
	mb.header["X-Emby-Authorization"] = mb.auth
	mb.Authenticated = true
	return user, resp.StatusCode, nil
}

// DeleteUser deletes the user corresponding to the provided ID.
func (mb *MediaBrowser) DeleteUser(userID string) (int, error) {
	if mb.serverType == JellyfinServer {
		return jfDeleteUser(mb, userID)
	}
	return embyDeleteUser(mb, userID)
}

// GetUsers returns all (visible) users on the Emby instance.
func (mb *MediaBrowser) GetUsers(public bool) ([]map[string]interface{}, int, error) {
	if mb.serverType == JellyfinServer {
		return jfGetUsers(mb, public)
	}
	return embyGetUsers(mb, public)
}

// UserByName returns the user corresponding to the provided username.
func (mb *MediaBrowser) UserByName(username string, public bool) (map[string]interface{}, int, error) {
	if mb.serverType == JellyfinServer {
		return jfUserByName(mb, username, public)
	}
	return embyUserByName(mb, username, public)
}

// UserByID returns the user corresponding to the provided ID.
func (mb *MediaBrowser) UserByID(userID string, public bool) (map[string]interface{}, int, error) {
	if mb.serverType == JellyfinServer {
		return jfUserByID(mb, userID, public)
	}
	return embyUserByID(mb, userID, public)
}

// NewUser creates a new user with the provided username and password.
func (mb *MediaBrowser) NewUser(username, password string) (map[string]interface{}, int, error) {
	if mb.serverType == JellyfinServer {
		return jfNewUser(mb, username, password)
	}
	return embyNewUser(mb, username, password)
}

// SetPolicy sets the access policy for the user corresponding to the provided ID.
func (mb *MediaBrowser) SetPolicy(userID string, policy map[string]interface{}) (int, error) {
	if mb.serverType == JellyfinServer {
		return jfSetPolicy(mb, userID, policy)
	}
	return embySetPolicy(mb, userID, policy)
}

// SetConfiguration sets the configuration (part of homescreen layout) for the user corresponding to the provided ID.
func (mb *MediaBrowser) SetConfiguration(userID string, configuration map[string]interface{}) (int, error) {
	if mb.serverType == JellyfinServer {
		return jfSetConfiguration(mb, userID, configuration)
	}
	return embySetConfiguration(mb, userID, configuration)
}

// GetDisplayPreferences gets the displayPreferences (part of homescreen layout) for the user corresponding to the provided ID.
func (mb *MediaBrowser) GetDisplayPreferences(userID string) (map[string]interface{}, int, error) {
	if mb.serverType == JellyfinServer {
		return jfGetDisplayPreferences(mb, userID)
	}
	return embyGetDisplayPreferences(mb, userID)
}

// SetDisplayPreferences sets the displayPreferences (part of homescreen layout) for the user corresponding to the provided ID.
func (mb *MediaBrowser) SetDisplayPreferences(userID string, displayprefs map[string]interface{}) (int, error) {
	if mb.serverType == JellyfinServer {
		return jfSetDisplayPreferences(mb, userID, displayprefs)
	}
	return embySetDisplayPreferences(mb, userID, displayprefs)
}
