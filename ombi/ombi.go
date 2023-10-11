package ombi

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hrfee/jfa-go/common"
)

const (
	NotifAgentDiscord  = 1
	NotifAgentTelegram = 4
)

// Ombi represents a running Ombi instance.
type Ombi struct {
	server, key    string
	header         map[string]string
	httpClient     *http.Client
	userCache      []map[string]interface{}
	cacheExpiry    time.Time
	cacheLength    int
	timeoutHandler common.TimeoutHandler
}

// NewOmbi returns an Ombi object.
func NewOmbi(server, key string, timeoutHandler common.TimeoutHandler) *Ombi {
	return &Ombi{
		server: server,
		key:    key,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		header: map[string]string{
			"ApiKey": key,
		},
		cacheLength:    30,
		cacheExpiry:    time.Now(),
		timeoutHandler: timeoutHandler,
	}
}

// does a GET and returns the response as a string.
func (ombi *Ombi) getJSON(url string, params map[string]string) (string, int, error) {
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
	defer ombi.timeoutHandler()
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
func (ombi *Ombi) send(mode string, url string, data interface{}, response bool, headers map[string]string) (string, int, error) {
	responseText := ""
	params, _ := json.Marshal(data)
	req, _ := http.NewRequest(mode, url, bytes.NewBuffer(params))
	req.Header.Add("Content-Type", "application/json")
	for name, value := range ombi.header {
		req.Header.Add(name, value)
	}
	for name, value := range headers {
		req.Header.Add(name, value)
	}
	resp, err := ombi.httpClient.Do(req)
	defer ombi.timeoutHandler()
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

func (ombi *Ombi) post(url string, data map[string]interface{}, response bool) (string, int, error) {
	return ombi.send("POST", url, data, response, nil)
}

func (ombi *Ombi) put(url string, data map[string]interface{}, response bool) (string, int, error) {
	return ombi.send("PUT", url, data, response, nil)
}

// ModifyUser applies the given modified user object to the corresponding user.
func (ombi *Ombi) ModifyUser(user map[string]interface{}) (status int, err error) {
	if _, ok := user["id"]; !ok {
		err = fmt.Errorf("No ID provided")
		return
	}
	_, status, err = ombi.put(ombi.server+"/api/v1/Identity/", user, false)
	return
}

// DeleteUser deletes the user corresponding to the given ID.
func (ombi *Ombi) DeleteUser(id string) (code int, err error) {
	url := fmt.Sprintf("%s/api/v1/Identity/%s", ombi.server, id)
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Add("Content-Type", "application/json")
	for name, value := range ombi.header {
		req.Header.Add(name, value)
	}
	resp, err := ombi.httpClient.Do(req)
	defer ombi.timeoutHandler()
	return resp.StatusCode, err
}

// UserByID returns the user corresponding to the provided ID.
func (ombi *Ombi) UserByID(id string) (result map[string]interface{}, code int, err error) {
	resp, code, err := ombi.getJSON(fmt.Sprintf("%s/api/v1/Identity/User/%s", ombi.server, id), nil)
	json.Unmarshal([]byte(resp), &result)
	return
}

// GetUsers returns all users on the Ombi instance.
func (ombi *Ombi) GetUsers() ([]map[string]interface{}, int, error) {
	if time.Now().After(ombi.cacheExpiry) {
		resp, code, err := ombi.getJSON(fmt.Sprintf("%s/api/v1/Identity/Users", ombi.server), nil)
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

// TemplateByID returns a template based on the user corresponding to the provided ID's settings.
func (ombi *Ombi) TemplateByID(id string) (result map[string]interface{}, code int, err error) {
	result, code, err = ombi.UserByID(id)
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

// NewUser creates a new user with the given username, password and email address.
func (ombi *Ombi) NewUser(username, password, email string, template map[string]interface{}) ([]string, int, error) {
	url := fmt.Sprintf("%s/api/v1/Identity", ombi.server)
	user := template
	user["userName"] = username
	user["password"] = password
	user["emailAddress"] = email
	resp, code, err := ombi.post(url, user, true)
	var data map[string]interface{}
	json.Unmarshal([]byte(resp), &data)
	if err != nil || code != 200 {
		var lst []string
		if data["errors"] != nil {
			lst = data["errors"].([]string)
		}
		return lst, code, err
	}
	ombi.cacheExpiry = time.Now()
	return nil, code, err
}

type NotificationPref struct {
	Agent   int    `json:"agent"`
	UserID  string `json:"userId"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

func (ombi *Ombi) SetNotificationPrefs(user map[string]interface{}, discordID, telegramUser string) (result string, code int, err error) {
	id := user["id"].(string)
	url := fmt.Sprintf("%s/api/v1/Identity/NotificationPreferences", ombi.server)
	data := []NotificationPref{}
	if discordID != "" {
		data = append(data, NotificationPref{NotifAgentDiscord, id, discordID, true})
	}
	if telegramUser != "" {
		data = append(data, NotificationPref{NotifAgentTelegram, id, telegramUser, true})
	}
	result, code, err = ombi.send("POST", url, data, true, map[string]string{"UserName": user["userName"].(string)})
	return
}
