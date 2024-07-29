package jellyseerr

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hrfee/jfa-go/common"
)

const (
	API_SUFFIX = "/api/v1"
)

// Jellyseerr represents a running Jellyseerr instance.
type Jellyseerr struct {
	server, key    string
	header         map[string]string
	httpClient     *http.Client
	userCache      map[string]User // Map of jellyfin IDs to users
	cacheExpiry    time.Time
	cacheLength    time.Duration
	timeoutHandler common.TimeoutHandler
}

// NewJellyseerr returns an Ombi object.
func NewJellyseerr(server, key string, timeoutHandler common.TimeoutHandler) *Jellyseerr {
	if !strings.HasSuffix(server, API_SUFFIX) {
		server = server + API_SUFFIX
	}
	return &Jellyseerr{
		server: server,
		key:    key,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		header: map[string]string{
			"X-Api-Key": key,
		},
		cacheLength:    time.Duration(30) * time.Minute,
		cacheExpiry:    time.Now(),
		timeoutHandler: timeoutHandler,
		userCache:      map[string]User{},
	}
}

// does a GET and returns the response as a string.
func (js *Jellyseerr) getJSON(url string, params map[string]string, queryParams url.Values) (string, int, error) {
	if js.key == "" {
		return "", 401, fmt.Errorf("No API key provided")
	}
	var req *http.Request
	if params != nil {
		jsonParams, _ := json.Marshal(params)
		req, _ = http.NewRequest("GET", url+"?"+queryParams.Encode(), bytes.NewBuffer(jsonParams))
	} else {
		req, _ = http.NewRequest("GET", url+"?"+queryParams.Encode(), nil)
	}
	for name, value := range js.header {
		req.Header.Add(name, value)
	}
	resp, err := js.httpClient.Do(req)
	defer js.timeoutHandler()
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
func (js *Jellyseerr) send(mode string, url string, data interface{}, response bool, headers map[string]string) (string, int, error) {
	responseText := ""
	params, _ := json.Marshal(data)
	req, _ := http.NewRequest(mode, url, bytes.NewBuffer(params))
	req.Header.Add("Content-Type", "application/json")
	for name, value := range js.header {
		req.Header.Add(name, value)
	}
	for name, value := range headers {
		req.Header.Add(name, value)
	}
	resp, err := js.httpClient.Do(req)
	defer js.timeoutHandler()
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

func (js *Jellyseerr) post(url string, data map[string]interface{}, response bool) (string, int, error) {
	return js.send("POST", url, data, response, nil)
}

func (js *Jellyseerr) put(url string, data map[string]interface{}, response bool) (string, int, error) {
	return js.send("PUT", url, data, response, nil)
}

func (js *Jellyseerr) ImportFromJellyfin(jfIDs ...string) ([]User, error) {
	params := map[string]interface{}{
		"jellyfinUserIds": jfIDs,
	}
	resp, status, err := js.post(js.server+"/user/import-from-jellyfin", params, true)
	var data []User
	if err != nil {
		return data, err
	}
	if status != 200 && status != 201 {
		return data, fmt.Errorf("failed (error %d)", status)
	}
	err = json.Unmarshal([]byte(resp), &data)
	for _, u := range data {
		if u.JellyfinUserID != "" {
			js.userCache[u.JellyfinUserID] = u
		}
	}
	return data, err
}

func (js *Jellyseerr) getUsers() error {
	if js.cacheExpiry.After(time.Now()) {
		return nil
	}
	js.cacheExpiry = time.Now().Add(js.cacheLength)
	pageCount := 1
	pageIndex := 0
	for {
		res, err := js.getUserPage(0)
		if err != nil {
			return err
		}
		for _, u := range res.Results {
			if u.JellyfinUserID == "" {
				continue
			}
			js.userCache[u.JellyfinUserID] = u
		}
		pageCount = res.Page.Pages
		pageIndex++
		if pageIndex >= pageCount {
			break
		}
	}
	return nil
}

func (js *Jellyseerr) getUserPage(page int) (GetUsersDTO, error) {
	params := url.Values{}
	params.Add("take", "30")
	params.Add("skip", strconv.Itoa(page))
	params.Add("sort", "created")
	resp, status, err := js.getJSON(js.server+"/user", nil, params)
	var data GetUsersDTO
	if status != 200 {
		return data, fmt.Errorf("failed (error %d)", status)
	}
	if err != nil {
		return data, err
	}
	err = json.Unmarshal([]byte(resp), &data)
	return data, err
}

// MustGetUser provides the same function as ImportFromJellyfin, but will always return the user,
// even if they already existed.
func (js *Jellyseerr) MustGetUser(jfID string) (User, error) {
	js.getUsers()
	if u, ok := js.userCache[jfID]; ok {
		return u, nil
	}
	users, err := js.ImportFromJellyfin(jfID)
	var u User
	if err != nil {
		return u, err
	}
	if len(users) != 0 {
		return users[0], err
	}
	if u, ok := js.userCache[jfID]; ok {
		return u, nil
	}
	return u, fmt.Errorf("user not found")
}

func (js *Jellyseerr) Me() (User, error) {
	resp, status, err := js.getJSON(js.server+"/auth/me", nil, url.Values{})
	var data User
	data.ID = -1
	if status != 200 {
		return data, fmt.Errorf("failed (error %d)", status)
	}
	if err != nil {
		return data, err
	}
	err = json.Unmarshal([]byte(resp), &data)
	return data, err
}
