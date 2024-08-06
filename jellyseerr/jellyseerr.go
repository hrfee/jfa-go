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

	co "github.com/hrfee/jfa-go/common"
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
	timeoutHandler   co.TimeoutHandler
	LogRequestBodies bool
	AutoImportUsers  bool
}

// NewJellyseerr returns an Ombi object.
func NewJellyseerr(server, key string, timeoutHandler co.TimeoutHandler) *Jellyseerr {
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

func (js *Jellyseerr) req(mode string, uri string, data any, queryParams url.Values, headers map[string]string, response bool) (string, int, error) {
	var params []byte
	if data != nil {
		params, _ = json.Marshal(data)
	}
	if js.LogRequestBodies {
		fmt.Printf("Jellyseerr API Client: Sending Data \"%s\" to \"%s\"\n", string(params), uri)
	}
	if qp := queryParams.Encode(); qp != "" {
		uri += "?" + qp
	}
	var req *http.Request
	if data != nil {
		req, _ = http.NewRequest(mode, uri, bytes.NewBuffer(params))
	} else {
		req, _ = http.NewRequest(mode, uri, nil)
	}
	req.Header.Add("Content-Type", "application/json")
	for name, value := range js.header {
		req.Header.Add(name, value)
	}
	if headers != nil {
		for name, value := range headers {
			req.Header.Add(name, value)
		}
	}
	resp, err := js.httpClient.Do(req)
	err = co.GenericErr(resp.StatusCode, err)
	defer js.timeoutHandler()
	var responseText string
	defer resp.Body.Close()
	if response || err != nil {
		responseText, err = js.decodeResp(resp)
		if err != nil {
			return responseText, resp.StatusCode, err
		}
	}
	if err != nil {
		var msg ErrorDTO
		err = json.Unmarshal([]byte(responseText), &msg)
		if err != nil {
			return responseText, resp.StatusCode, err
		}
		if msg.Message != "" {
			err = fmt.Errorf("got %d: %s", resp.StatusCode, msg.Message)
		}
		return responseText, resp.StatusCode, err
	}
	return responseText, resp.StatusCode, err
}

func (js *Jellyseerr) decodeResp(resp *http.Response) (string, error) {
	var out io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		out, _ = gzip.NewReader(resp.Body)
	default:
		out = resp.Body
	}
	buf := new(strings.Builder)
	_, err := io.Copy(buf, out)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (js *Jellyseerr) get(uri string, data any, params url.Values) (string, int, error) {
	return js.req(http.MethodGet, uri, data, params, nil, true)
}

func (js *Jellyseerr) post(uri string, data any, response bool) (string, int, error) {
	return js.req(http.MethodPost, uri, data, url.Values{}, nil, response)
}

func (js *Jellyseerr) put(uri string, data any, response bool) (string, int, error) {
	return js.req(http.MethodPut, uri, data, url.Values{}, nil, response)
}

func (js *Jellyseerr) delete(uri string, data any) (int, error) {
	_, status, err := js.req(http.MethodDelete, uri, data, url.Values{}, nil, false)
	return status, err
}

func (js *Jellyseerr) ImportFromJellyfin(jfIDs ...string) ([]User, error) {
	params := map[string]interface{}{
		"jellyfinUserIds": jfIDs,
	}
	resp, _, err := js.post(js.server+"/user/import-from-jellyfin", params, true)
	var data []User
	if err != nil {
		return data, err
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
	resp, _, err := js.get(js.server+"/user", nil, params)
	var data GetUsersDTO
	if err == nil {
		err = json.Unmarshal([]byte(resp), &data)
	}
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
	resp, _, err := js.get(js.server+"/auth/me", nil, url.Values{})
	var data User
	data.ID = -1
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

	resp, _, err := js.get(fmt.Sprintf(js.server+"/user/%d/settings/permissions", u.ID), nil, url.Values{})
	if err != nil {
		return data.Permissions, err
	}
	err = json.Unmarshal([]byte(resp), &data)
	return data.Permissions, err
}

func (js *Jellyseerr) SetPermissions(jfID string, perm Permissions) error {
	u, err := js.getUser(jfID)
	if err != nil {
		return err
	}

	_, _, err = js.post(fmt.Sprintf(js.server+"/user/%d/settings/permissions", u.ID), permissionsDTO{Permissions: perm}, false)
	if err != nil {
		return err
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

	_, _, err = js.put(fmt.Sprintf(js.server+"/user/%d", u.ID), tmpl, false)
	if err != nil {
		return err
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

	_, _, err = js.put(fmt.Sprintf(js.server+"/user/%d", u.ID), conf, false)
	if err != nil {
		return err
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

	_, err = js.delete(fmt.Sprintf(js.server+"/user/%d", u.ID), nil)
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
	resp, _, err := js.get(fmt.Sprintf(js.server+"/user/%d/settings/notifications", jellyseerrID), nil, url.Values{})
	if err != nil {
		return data, err
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

	_, _, err = js.post(fmt.Sprintf(js.server+"/user/%d/settings/notifications", u.ID), tmpl, false)
	if err != nil {
		return err
	}
	return nil
}

func (js *Jellyseerr) ModifyNotifications(jfID string, conf map[NotificationsField]any) error {
	u, err := js.getUser(jfID)
	if err != nil {
		return err
	}

	_, _, err = js.post(fmt.Sprintf(js.server+"/user/%d/settings/notifications", u.ID), conf, false)
	if err != nil {
		return err
	}
	return nil
}

func (js *Jellyseerr) GetUsers() (map[string]User, error) {
	err := js.getUsers()
	return js.userCache, err
}

func (js *Jellyseerr) UserByID(jellyseerrID int64) (User, error) {
	resp, _, err := js.get(js.server+fmt.Sprintf("/user/%d", jellyseerrID), nil, url.Values{})
	var data User
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

	_, _, err = js.post(fmt.Sprintf(js.server+"/user/%d/settings/main", u.ID), conf, false)
	if err != nil {
		return err
	}
	// Lazily just invalidate the cache.
	js.cacheExpiry = time.Now()
	return nil
}
