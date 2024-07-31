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
	API_SUFFIX      = "/api/v1"
	BogusIdentifier = "123412341234123456"
)

// Jellyseerr represents a running Jellyseerr instance.
type Jellyseerr struct {
	server, key      string
	header           map[string]string
	httpClient       *http.Client
	userCache        map[string]User // Map of jellyfin IDs to users
	cacheExpiry      time.Time
	cacheLength      time.Duration
	timeoutHandler   common.TimeoutHandler
	LogRequestBodies bool
	AutoImportUsers  bool
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
		cacheLength:      time.Duration(30) * time.Minute,
		cacheExpiry:      time.Now(),
		timeoutHandler:   timeoutHandler,
		userCache:        map[string]User{},
		LogRequestBodies: false,
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
		if js.LogRequestBodies {
			fmt.Printf("Jellyseerr API Client: Sending Data \"%s\" to \"%s\"\n", string(jsonParams), url)
		}
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
func (js *Jellyseerr) send(mode string, url string, data any, response bool, headers map[string]string) (string, int, error) {
	responseText := ""
	params, _ := json.Marshal(data)
	if js.LogRequestBodies {
		fmt.Printf("Jellyseerr API Client: Sending Data \"%s\" to \"%s\"\n", string(params), url)
	}
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

func (js *Jellyseerr) post(url string, data any, response bool) (string, int, error) {
	return js.send("POST", url, data, response, nil)
}

func (js *Jellyseerr) put(url string, data any, response bool) (string, int, error) {
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
		res, err := js.getUserPage(pageIndex)
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
	params.Add("skip", strconv.Itoa(page*30))
	params.Add("sort", "created")
	if js.LogRequestBodies {
		fmt.Printf("Jellyseerr API Client: Sending with URL params \"%+v\"\n", params)
	}
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

func (js *Jellyseerr) MustGetUser(jfID string) (User, error) {
	u, _, err := js.GetOrImportUser(jfID)
	return u, err
}

// GetImportedUser provides the same function as ImportFromJellyfin, but will always return the user,
// even if they already existed. Also returns whether the user was imported or not,
func (js *Jellyseerr) GetOrImportUser(jfID string) (u User, imported bool, err error) {
	imported = false
	u, err = js.GetExistingUser(jfID)
	if err == nil {
		return
	}
	var users []User
	users, err = js.ImportFromJellyfin(jfID)
	if err != nil {
		return
	}
	if len(users) != 0 {
		u = users[0]
		err = nil
		return
	}
	err = fmt.Errorf("user not found or imported")
	return
}

func (js *Jellyseerr) GetExistingUser(jfID string) (u User, err error) {
	js.getUsers()
	ok := false
	err = nil
	if u, ok = js.userCache[jfID]; ok {
		return
	}
	js.cacheExpiry = time.Now()
	js.getUsers()
	if u, ok = js.userCache[jfID]; ok {
		err = nil
		return
	}
	err = fmt.Errorf("user not found")
	return
}

func (js *Jellyseerr) getUser(jfID string) (User, error) {
	if js.AutoImportUsers {
		return js.MustGetUser(jfID)
	}
	return js.GetExistingUser(jfID)
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

func (js *Jellyseerr) GetPermissions(jfID string) (Permissions, error) {
	data := permissionsDTO{Permissions: -1}
	u, err := js.getUser(jfID)
	if err != nil {
		return data.Permissions, err
	}

	resp, status, err := js.getJSON(fmt.Sprintf(js.server+"/user/%d/settings/permissions", u.ID), nil, url.Values{})
	if err != nil {
		return data.Permissions, err
	}
	if status != 200 {
		return data.Permissions, fmt.Errorf("failed (error %d)", status)
	}
	err = json.Unmarshal([]byte(resp), &data)
	return data.Permissions, err
}

func (js *Jellyseerr) SetPermissions(jfID string, perm Permissions) error {
	u, err := js.getUser(jfID)
	if err != nil {
		return err
	}

	_, status, err := js.post(fmt.Sprintf(js.server+"/user/%d/settings/permissions", u.ID), permissionsDTO{Permissions: perm}, false)
	if err != nil {
		return err
	}
	if status != 200 && status != 201 {
		return fmt.Errorf("failed (error %d)", status)
	}
	u.Permissions = perm
	js.userCache[jfID] = u
	return nil
}

func (js *Jellyseerr) ApplyTemplateToUser(jfID string, tmpl UserTemplate) error {
	u, err := js.getUser(jfID)
	if err != nil {
		return err
	}

	_, status, err := js.put(fmt.Sprintf(js.server+"/user/%d", u.ID), tmpl, false)
	if err != nil {
		return err
	}
	if status != 200 && status != 201 {
		return fmt.Errorf("failed (error %d)", status)
	}
	u.UserTemplate = tmpl
	js.userCache[jfID] = u
	return nil
}

func (js *Jellyseerr) ModifyUser(jfID string, conf map[UserField]any) error {
	if _, ok := conf[FieldEmail]; ok {
		return fmt.Errorf("email is read only, set with ModifyMainUserSettings instead")
	}
	u, err := js.getUser(jfID)
	if err != nil {
		return err
	}

	_, status, err := js.put(fmt.Sprintf(js.server+"/user/%d", u.ID), conf, false)
	if err != nil {
		return err
	}
	if status != 200 && status != 201 {
		return fmt.Errorf("failed (error %d)", status)
	}
	// Lazily just invalidate the cache.
	js.cacheExpiry = time.Now()
	return nil
}

func (js *Jellyseerr) DeleteUser(jfID string) error {
	u, err := js.getUser(jfID)
	if err != nil {
		return err
	}

	_, status, err := js.send("DELETE", fmt.Sprintf(js.server+"/user/%d", u.ID), nil, false, nil)
	if status != 200 && status != 201 {
		return fmt.Errorf("failed (error %d)", status)
	}
	if err != nil {
		return err
	}
	delete(js.userCache, jfID)
	return err
}

func (js *Jellyseerr) GetNotificationPreferences(jfID string) (Notifications, error) {
	u, err := js.getUser(jfID)
	if err != nil {
		return Notifications{}, err
	}
	return js.GetNotificationPreferencesByID(u.ID)
}

func (js *Jellyseerr) GetNotificationPreferencesByID(jellyseerrID int64) (Notifications, error) {
	var data Notifications
	resp, status, err := js.getJSON(fmt.Sprintf(js.server+"/user/%d/settings/notifications", jellyseerrID), nil, url.Values{})
	if err != nil {
		return data, err
	}
	if status != 200 {
		return data, fmt.Errorf("failed (error %d)", status)
	}
	err = json.Unmarshal([]byte(resp), &data)
	return data, err
}

func (js *Jellyseerr) ApplyNotificationsTemplateToUser(jfID string, tmpl NotificationsTemplate) error {
	// This behaviour is not desired, this being all-zero means no notifications, which is a settings state we'd want to store!
	/* if tmpl.NotifTypes.Empty() {
		tmpl.NotifTypes = nil
	}*/
	u, err := js.getUser(jfID)
	if err != nil {
		return err
	}

	_, status, err := js.post(fmt.Sprintf(js.server+"/user/%d/settings/notifications", u.ID), tmpl, false)
	if err != nil {
		return err
	}
	if status != 200 && status != 201 {
		return fmt.Errorf("failed (error %d)", status)
	}
	return nil
}

func (js *Jellyseerr) ModifyNotifications(jfID string, conf map[NotificationsField]any) error {
	u, err := js.getUser(jfID)
	if err != nil {
		return err
	}

	_, status, err := js.post(fmt.Sprintf(js.server+"/user/%d/settings/notifications", u.ID), conf, false)
	if err != nil {
		return err
	}
	if status != 200 && status != 201 {
		return fmt.Errorf("failed (error %d)", status)
	}
	return nil
}

func (js *Jellyseerr) GetUsers() (map[string]User, error) {
	err := js.getUsers()
	return js.userCache, err
}

func (js *Jellyseerr) UserByID(jellyseerrID int64) (User, error) {
	resp, status, err := js.getJSON(js.server+fmt.Sprintf("/user/%d", jellyseerrID), nil, url.Values{})
	var data User
	if status != 200 {
		return data, fmt.Errorf("failed (error %d)", status)
	}
	if err != nil {
		return data, err
	}
	err = json.Unmarshal([]byte(resp), &data)
	return data, err
}

func (js *Jellyseerr) ModifyMainUserSettings(jfID string, conf MainUserSettings) error {
	u, err := js.getUser(jfID)
	if err != nil {
		return err
	}

	_, status, err := js.post(fmt.Sprintf(js.server+"/user/%d/settings/main", u.ID), conf, false)
	if err != nil {
		return err
	}
	if status != 200 && status != 201 {
		return fmt.Errorf("failed (error %d)", status)
	}
	// Lazily just invalidate the cache.
	js.cacheExpiry = time.Now()
	return nil
}
