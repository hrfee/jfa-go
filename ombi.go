package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Ombi struct {
	server, key string
	header      map[string]string
	httpClient  *http.Client
	noFail      bool
	userCache   []map[string]interface{}
	cacheExpiry time.Time
	cacheLength int
}

func newOmbi(server, key string, noFail bool) *Ombi {
	return &Ombi{
		server: server,
		key:    key,
		noFail: noFail,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		header: map[string]string{
			"ApiKey": key,
		},
		cacheLength: 30,
		cacheExpiry: time.Now(),
	}
}

// does a GET and returns the response as a string.
func (ombi *Ombi) _getJSON(url string, params map[string]string) (string, int, error) {
	if ombi.key == "" {
		return "", 401, fmt.Errorf("No API key provided")
	}
	var req *http.Request
	if params != nil {
		jsonParams, _ := json.Marshal(params)
		req, _ = http.NewRequest("GET", url, bytes.NewBuffer(jsonParams))
	} else {
		req, _ = http.NewRequest("GET", url, nil)
	}
	for name, value := range ombi.header {
		req.Header.Add(name, value)
	}
	resp, err := ombi.httpClient.Do(req)
	defer timeoutHandler("Ombi", ombi.server, ombi.noFail)
	if err != nil || resp.StatusCode != 200 {
		if resp.StatusCode == 401 {
			return "", 401, fmt.Errorf("Invalid API Key")
		}
		return "", resp.StatusCode, err
	}
	defer resp.Body.Close()
	var data io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		data, _ = gzip.NewReader(resp.Body)
	default:
		data = resp.Body
	}
	buf := new(strings.Builder)
	_, err = io.Copy(buf, data)
	if err != nil {
		return "", 500, err
	}
	return buf.String(), resp.StatusCode, nil
}

// does a POST and optionally returns response as string. Returns a string instead of an io.reader bcs i couldn't get it working otherwise.
func (ombi *Ombi) _send(mode string, url string, data map[string]interface{}, response bool) (string, int, error) {
	responseText := ""
	params, _ := json.Marshal(data)
	req, _ := http.NewRequest(mode, url, bytes.NewBuffer(params))
	req.Header.Add("Content-Type", "application/json")
	for name, value := range ombi.header {
		req.Header.Add(name, value)
	}
	resp, err := ombi.httpClient.Do(req)
	defer timeoutHandler("Ombi", ombi.server, ombi.noFail)
	if err != nil || !(resp.StatusCode == 200 || resp.StatusCode == 201) {
		if resp.StatusCode == 401 {
			return "", 401, fmt.Errorf("Invalid API Key")
		}
		return responseText, resp.StatusCode, err
	}
	if response {
		defer resp.Body.Close()
		var out io.Reader
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			out, _ = gzip.NewReader(resp.Body)
		default:
			out = resp.Body
		}
		buf := new(strings.Builder)
		_, err = io.Copy(buf, out)
		if err != nil {
			return "", 500, err
		}
		responseText = buf.String()
	}
	return responseText, resp.StatusCode, nil
}

func (ombi *Ombi) _post(url string, data map[string]interface{}, response bool) (string, int, error) {
	return ombi._send("POST", url, data, response)
}

func (ombi *Ombi) _put(url string, data map[string]interface{}, response bool) (string, int, error) {
	return ombi._send("PUT", url, data, response)
}

func (ombi *Ombi) modifyUser(user map[string]interface{}) (status int, err error) {
	if _, ok := user["id"]; !ok {
		err = fmt.Errorf("No ID provided")
		return
	}
	_, status, err = ombi._put(ombi.server+"/api/v1/Identity", user, false)
	return
}

func (ombi *Ombi) deleteUser(id string) (code int, err error) {
	url := fmt.Sprintf("%s/api/v1/Identity/%s", ombi.server, id)
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Add("Content-Type", "application/json")
	for name, value := range ombi.header {
		req.Header.Add(name, value)
	}
	resp, err := ombi.httpClient.Do(req)
	defer timeoutHandler("Ombi", ombi.server, ombi.noFail)
	return resp.StatusCode, err
}

// gets an ombi user by their ID.
func (ombi *Ombi) userByID(id string) (result map[string]interface{}, code int, err error) {
	resp, code, err := ombi._getJSON(fmt.Sprintf("%s/api/v1/Identity/User/%s", ombi.server, id), nil)
	json.Unmarshal([]byte(resp), &result)
	return
}

// gets a list of all users.
func (ombi *Ombi) getUsers() ([]map[string]interface{}, int, error) {
	if time.Now().After(ombi.cacheExpiry) {
		resp, code, err := ombi._getJSON(fmt.Sprintf("%s/api/v1/Identity/Users", ombi.server), nil)
		var result []map[string]interface{}
		json.Unmarshal([]byte(resp), &result)
		ombi.userCache = result
		if (code == 200 || code == 204) && err == nil {
			ombi.cacheExpiry = time.Now().Add(time.Minute * time.Duration(ombi.cacheLength))
		}
		return result, code, err
	}
	return ombi.userCache, 200, nil
}

// Strip these from a user when saving as a template.
// We also need to strip userQualityProfiles{"id", "userId"}
var stripFromOmbi = []string{
	"alias",
	"emailAddress",
	"hasLoggedIn",
	"id",
	"lastLoggedIn",
	"password",
	"userName",
}

// returns a template based on the user corresponding to the provided ID's settings.
func (ombi *Ombi) templateByID(id string) (result map[string]interface{}, code int, err error) {
	result, code, err = ombi.userByID(id)
	if err != nil || code != 200 {
		return
	}
	for _, key := range stripFromOmbi {
		if _, ok := result[key]; ok {
			delete(result, key)
		}
	}
	if qp, ok := result["userQualityProfiles"].(map[string]interface{}); ok {
		delete(qp, "id")
		delete(qp, "userId")
		result["userQualityProfiles"] = qp
	}
	return
}

// creates a new user.
func (ombi *Ombi) newUser(username, password, email string, template map[string]interface{}) ([]string, int, error) {
	url := fmt.Sprintf("%s/api/v1/Identity", ombi.server)
	user := template
	user["userName"] = username
	user["password"] = password
	user["emailAddress"] = email
	resp, code, err := ombi._post(url, user, true)
	var data map[string]interface{}
	json.Unmarshal([]byte(resp), &data)
	if err != nil || code != 200 {
		var lst []string
		if data["errors"] != nil {
			lst = data["errors"].([]string)
		}
		return lst, code, err
	}
	return nil, code, err
}
