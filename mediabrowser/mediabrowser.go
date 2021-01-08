package mediabrowser

import (
	"net/http"
	"time"
)

type serverInfo struct {
	LocalAddress string `json:"LocalAddress"`
	Name         string `json:"ServerName"`
	Version      string `json:"Version"`
	OS           string `json:"OperatingSystem"`
	ID           string `json:"Id"`
}

type MediaBrowserStruct struct {
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
	timeoutHandler TimeoutHandler
}

// MediaBrowser is an api instance of Jellyfin/Emby.
type MediaBrowser interface {
	Authenticate(username, password string) (map[string]interface{}, int, error)
	DeleteUser(userID string) (int, error)
	GetUsers(public bool) ([]map[string]interface{}, int, error)
	UserByName(username string, public bool) (map[string]interface{}, int, error)
	UserByID(userID string, public bool) (map[string]interface{}, int, error)
	NewUser(username, password string) (map[string]interface{}, int, error)
	SetPolicy(userID string, policy map[string]interface{}) (int, error)
	SetConfiguration(userID string, configuration map[string]interface{}) (int, error)
	GetDisplayPreferences(userID string) (map[string]interface{}, int, error)
	SetDisplayPreferences(userID string, displayprefs map[string]interface{}) (int, error)
}
