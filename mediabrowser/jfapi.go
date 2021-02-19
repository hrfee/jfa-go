package mediabrowser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func jfDeleteUser(jf *MediaBrowser, userID string) (int, error) {
	url := fmt.Sprintf("%s/Users/%s", jf.Server, userID)
	req, _ := http.NewRequest("DELETE", url, nil)
	for name, value := range jf.header {
		req.Header.Add(name, value)
	}
	resp, err := jf.httpClient.Do(req)
	defer jf.timeoutHandler()
	return resp.StatusCode, err
}

func jfGetUsers(jf *MediaBrowser, public bool) ([]User, int, error) {
	var result []User
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
		err := json.Unmarshal([]byte(data), &result)
		if err != nil {
			fmt.Println(err)
			return nil, status, err
		}
		jf.userCache = result
		jf.CacheExpiry = time.Now().Add(time.Minute * time.Duration(jf.cacheLength))
		if result[0].ID[8] == '-' {
			jf.Hyphens = true
		}
		return result, status, nil
	}
	return jf.userCache, 200, nil
}

func jfUserByName(jf *MediaBrowser, username string, public bool) (User, int, error) {
	var match User
	find := func() (User, int, error) {
		users, status, err := jf.GetUsers(public)
		if err != nil || status != 200 {
			return User{}, status, err
		}
		for _, user := range users {
			if user.Name == username {
				return user, status, err
			}
		}
		return User{}, status, err
	}
	match, status, err := find()
	if match.Name == "" {
		jf.CacheExpiry = time.Now()
		match, status, err = find()
	}
	return match, status, err
}

func jfUserByID(jf *MediaBrowser, userID string, public bool) (User, int, error) {
	if jf.CacheExpiry.After(time.Now()) {
		for _, user := range jf.userCache {
			if user.ID == userID {
				return user, 200, nil
			}
		}
	}
	if public {
		users, status, err := jf.GetUsers(public)
		if err != nil || status != 200 {
			return User{}, status, err
		}
		for _, user := range users {
			if user.ID == userID {
				return user, status, nil
			}
		}
		return User{}, status, err
	}
	var result User
	var data string
	var status int
	var err error
	url := fmt.Sprintf("%s/users/%s", jf.Server, userID)
	data, status, err = jf.get(url, jf.loginParams)
	if err != nil || status != 200 {
		return User{}, status, err
	}
	json.Unmarshal([]byte(data), &result)
	return result, status, nil
}

func jfNewUser(jf *MediaBrowser, username, password string) (User, int, error) {
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
	var recv User
	json.Unmarshal([]byte(response), &recv)
	if err != nil || !(status == 200 || status == 204) {
		return User{}, status, err
	}
	return recv, status, nil
}

func jfSetPolicy(jf *MediaBrowser, userID string, policy Policy) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Policy", jf.Server, userID)
	_, status, err := jf.post(url, policy, false)
	if err != nil || status != 200 {
		return status, err
	}
	return status, nil
}

func jfSetConfiguration(jf *MediaBrowser, userID string, configuration Configuration) (int, error) {
	url := fmt.Sprintf("%s/Users/%s/Configuration", jf.Server, userID)
	_, status, err := jf.post(url, configuration, false)
	return status, err
}

func jfGetDisplayPreferences(jf *MediaBrowser, userID string) (map[string]interface{}, int, error) {
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

func jfSetDisplayPreferences(jf *MediaBrowser, userID string, displayprefs map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/DisplayPreferences/usersettings?userId=%s&client=emby", jf.Server, userID)
	_, status, err := jf.post(url, displayprefs, false)
	if err != nil || !(status == 204 || status == 200) {
		return status, err
	}
	return status, nil
}
