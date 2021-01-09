package mediabrowser

// Almost identical to jfapi, with the most notable change being the password workaround.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func embyDeleteUser(emby *MediaBrowser, userID string) (int, error) {
	url := fmt.Sprintf("%s/Users/%s", emby.Server, userID)
	req, _ := http.NewRequest("DELETE", url, nil)
	for name, value := range emby.header {
		req.Header.Add(name, value)
	}
	resp, err := emby.httpClient.Do(req)
	defer emby.timeoutHandler()
	return resp.StatusCode, err
}

func embyGetUsers(emby *MediaBrowser, public bool) ([]map[string]interface{}, int, error) {
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

func embyUserByName(emby *MediaBrowser, username string, public bool) (map[string]interface{}, int, error) {
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

func embyUserByID(emby *MediaBrowser, userID string, public bool) (map[string]interface{}, int, error) {
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

// Since emby doesn't allow one to specify a password on user creation, we:
// Create the account
// Immediately disable it
// Set password
// Reeenable it
func embyNewUser(emby *MediaBrowser, username, password string) (map[string]interface{}, int, error) {
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
	url = fmt.Sprintf("%s/Users/%s/Password", emby.Server, id)
	data = map[string]interface{}{
		"Id":        id,
		"CurrentPw": "",
		"NewPw":     password,
	}
	_, status, err = emby.post(url, data, false)
	// Step 3: If setting password errored, try to delete the account
	if err != nil || !(status == 200 || status == 204) {
		_, err = emby.DeleteUser(id)
	}
	return recv, status, nil
}

func embySetPolicy(emby *MediaBrowser, userID string, policy map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Policy", emby.Server, userID)
	_, status, err := emby.post(url, policy, false)
	if err != nil || status != 200 {
		return status, err
	}
	return status, nil
}

func embySetConfiguration(emby *MediaBrowser, userID string, configuration map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Configuration", emby.Server, userID)
	_, status, err := emby.post(url, configuration, false)
	return status, err
}

func embyGetDisplayPreferences(emby *MediaBrowser, userID string) (map[string]interface{}, int, error) {
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

func embySetDisplayPreferences(emby *MediaBrowser, userID string, displayprefs map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/DisplayPreferences/usersettings?userId=%s&client=emby", emby.Server, userID)
	_, status, err := emby.post(url, displayprefs, false)
	if err != nil || !(status == 204 || status == 200) {
		return status, err
	}
	return status, nil
}
