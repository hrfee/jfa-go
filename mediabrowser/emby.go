package mediabrowser

// Almost identical to jfapi, with the most notable change being the password workaround.

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

// NewEmby returns a new Emby object.
func NewEmby(server, client, version, device, deviceID string, timeoutHandler common.TimeoutHandler, cacheTimeout int) (*MediaBrowserStruct, error) {
	emby := &Emby{}
	emby.Server = server
	emby.client = client
	emby.version = version
	emby.device = device
	emby.deviceID = deviceID
	emby.useragent = fmt.Sprintf("%s/%s", client, version)
	emby.timeoutHandler = timeoutHandler
	emby.auth = fmt.Sprintf("MediaBrowser Client=\"%s\", Device=\"%s\", DeviceId=\"%s\", Version=\"%s\"", client, device, deviceID, version)
	emby.header = map[string]string{
		"Accept":               "application/json",
		"Content-type":         "application/json; charset=UTF-8",
		"X-Application":        emby.useragent,
		"Accept-Charset":       "UTF-8,*",
		"Accept-Encoding":      "gzip",
		"User-Agent":           emby.useragent,
		"X-Emby-Authorization": emby.auth,
	}
	emby.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	infoURL := fmt.Sprintf("%s/System/Info/Public", server)
	req, _ := http.NewRequest("GET", infoURL, nil)
	resp, err := emby.httpClient.Do(req)
	defer emby.timeoutHandler()
	if err == nil {
		data, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(data, &emby.ServerInfo)
	}
	emby.cacheLength = cacheTimeout
	emby.CacheExpiry = time.Now()
	return emby, nil
}

// Authenticate attempts to authenticate using a username & password
func (emby *MediaBrowserStruct) Authenticate(username, password string) (map[string]interface{}, int, error) {
	emby.Username = username
	emby.password = password
	emby.loginParams = map[string]string{
		"Username": username,
		"Pw":       password,
		"Password": password,
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(emby.loginParams)
	if err != nil {
		return nil, 0, err
	}
	// loginParams, _ := json.Marshal(jf.loginParams)
	url := fmt.Sprintf("%s/Users/authenticatebyname", emby.Server)
	req, err := http.NewRequest("POST", url, buffer)
	defer emby.timeoutHandler()
	if err != nil {
		return nil, 0, err
	}
	for name, value := range emby.header {
		req.Header.Add(name, value)
	}
	resp, err := emby.httpClient.Do(req)
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
	emby.AccessToken = respData["AccessToken"].(string)
	user := respData["User"].(map[string]interface{})
	emby.userID = respData["User"].(map[string]interface{})["Id"].(string)
	emby.auth = fmt.Sprintf("MediaBrowser Client=\"%s\", Device=\"%s\", DeviceId=\"%s\", Version=\"%s\", Token=\"%s\"", emby.client, emby.device, emby.deviceID, emby.version, emby.AccessToken)
	emby.header["X-Emby-Authorization"] = emby.auth
	emby.Authenticated = true
	return user, resp.StatusCode, nil
}

func (emby *MediaBrowserStruct) get(url string, params map[string]string) (string, int, error) {
	var req *http.Request
	if params != nil {
		jsonParams, _ := json.Marshal(params)
		req, _ = http.NewRequest("GET", url, bytes.NewBuffer(jsonParams))
	} else {
		req, _ = http.NewRequest("GET", url, nil)
	}
	for name, value := range emby.header {
		req.Header.Add(name, value)
	}
	resp, err := emby.httpClient.Do(req)
	defer emby.timeoutHandler()
	if err != nil || resp.StatusCode != 200 {
		if resp.StatusCode == 401 && emby.Authenticated {
			emby.Authenticated = false
			_, _, authErr := emby.Authenticate(emby.Username, emby.password)
			if authErr == nil {
				v1, v2, v3 := emby.get(url, params)
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

func (emby *MediaBrowserStruct) post(url string, data map[string]interface{}, response bool) (string, int, error) {
	params, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(params))
	for name, value := range emby.header {
		req.Header.Add(name, value)
	}
	resp, err := emby.httpClient.Do(req)
	defer emby.timeoutHandler()
	if err != nil || resp.StatusCode != 200 {
		if resp.StatusCode == 401 && emby.Authenticated {
			emby.Authenticated = false
			_, _, authErr := emby.Authenticate(emby.Username, emby.password)
			if authErr == nil {
				v1, v2, v3 := emby.post(url, data, response)
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
func (emby *MediaBrowserStruct) DeleteUser(userID string) (int, error) {
	url := fmt.Sprintf("%s/Users/%s", emby.Server, userID)
	req, _ := http.NewRequest("DELETE", url, nil)
	for name, value := range emby.header {
		req.Header.Add(name, value)
	}
	resp, err := emby.httpClient.Do(req)
	defer emby.timeoutHandler()
	return resp.StatusCode, err
}

// GetUsers returns all (visible) users on the Emby instance.
func (emby *MediaBrowserStruct) GetUsers(public bool) ([]map[string]interface{}, int, error) {
	var result []map[string]interface{}
	var data string
	var status int
	var err error
	if time.Now().After(emby.CacheExpiry) {
		if public {
			url := fmt.Sprintf("%s/users/public", emby.Server)
			data, status, err = emby.get(url, nil)
		} else {
			url := fmt.Sprintf("%s/users", emby.Server)
			data, status, err = emby.get(url, emby.loginParams)
		}
		if err != nil || status != 200 {
			return nil, status, err
		}
		json.Unmarshal([]byte(data), &result)
		emby.userCache = result
		emby.CacheExpiry = time.Now().Add(time.Minute * time.Duration(emby.cacheLength))
		if id, ok := result[0]["Id"]; ok {
			if id.(string)[8] == '-' {
				emby.Hyphens = true
			}
		}
		return result, status, nil
	}
	return emby.userCache, 200, nil
}

// UserByName returns the user corresponding to the provided username.
func (emby *MediaBrowserStruct) UserByName(username string, public bool) (map[string]interface{}, int, error) {
	var match map[string]interface{}
	find := func() (map[string]interface{}, int, error) {
		users, status, err := emby.GetUsers(public)
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
		emby.CacheExpiry = time.Now()
		match, status, err = find()
	}
	return match, status, err
}

// UserByID returns the user corresponding to the provided ID.
func (emby *MediaBrowserStruct) UserByID(userID string, public bool) (map[string]interface{}, int, error) {
	if emby.CacheExpiry.After(time.Now()) {
		for _, user := range emby.userCache {
			if user["Id"].(string) == userID {
				return user, 200, nil
			}
		}
	}
	if public {
		users, status, err := emby.GetUsers(public)
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
	url := fmt.Sprintf("%s/users/%s", emby.Server, userID)
	data, status, err = emby.get(url, emby.loginParams)
	if err != nil || status != 200 {
		return nil, status, err
	}
	json.Unmarshal([]byte(data), &result)
	return result, status, nil
}

// NewUser creates a new user with the provided username and password.
// Since emby doesn't allow one to specify a password on user creation, we:
// Create the account
// Immediately disable it
// Set password
// Reeenable it
func (emby *MediaBrowserStruct) NewUser(username, password string) (map[string]interface{}, int, error) {
	url := fmt.Sprintf("%s/Users/New", emby.Server)
	data := map[string]interface{}{
		"Name": username,
	}
	response, status, err := emby.post(url, data, true)
	var recv map[string]interface{}
	json.Unmarshal([]byte(response), &recv)
	if err != nil || !(status == 200 || status == 204) {
		return nil, status, err
	}
	// Step 2: Set password
	id := recv["Id"].(string)
	url = fmt.Sprintf("/Users/%s/Password", id)
	data = map[string]interface{}{
		"Id":        id,
		"CurrentPw": "",
		"NewPw":     password,
	}
	_, status, err = emby.post(url, data, false)
	// Step 3: If setting password errored, try to delete the account
	if err != nil || !(status == 200 || status == 204) {
		status, err = emby.DeleteUser(id)
	}
	return recv, status, nil
}

// SetPolicy sets the access policy for the user corresponding to the provided ID.
func (emby *MediaBrowserStruct) SetPolicy(userID string, policy map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Policy", emby.Server, userID)
	_, status, err := emby.post(url, policy, false)
	if err != nil || status != 200 {
		return status, err
	}
	return status, nil
}

// SetConfiguration sets the configuration (part of homescreen layout) for the user corresponding to the provided ID.
func (emby *MediaBrowserStruct) SetConfiguration(userID string, configuration map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Configuration", emby.Server, userID)
	_, status, err := emby.post(url, configuration, false)
	return status, err
}

// GetDisplayPreferences gets the displayPreferences (part of homescreen layout) for the user corresponding to the provided ID.
func (emby *MediaBrowserStruct) GetDisplayPreferences(userID string) (map[string]interface{}, int, error) {
	url := fmt.Sprintf("%s/DisplayPreferences/usersettings?userId=%s&client=emby", emby.Server, userID)
	data, status, err := emby.get(url, nil)
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
func (emby *MediaBrowserStruct) SetDisplayPreferences(userID string, displayprefs map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/DisplayPreferences/usersettings?userId=%s&client=emby", emby.Server, userID)
	_, status, err := emby.post(url, displayprefs, false)
	if err != nil || !(status == 204 || status == 200) {
		return status, err
	}
	return status, nil
}
