package main

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/easyproxy"
	"github.com/hrfee/mediabrowser"
)

func (app *appContext) ServeSetup(gc *gin.Context) {
	lang := gc.Query("lang")
	if lang == "" {
		lang = "en-us"
	} else if _, ok := app.storage.lang.Admin[lang]; !ok {
		lang = "en-us"
	}
	emailLang := lang
	if _, ok := app.storage.lang.Email[lang]; !ok {
		emailLang = "en-us"
	}

	messages := map[string]map[string]string{
		"ui": {
			"contact_message": app.config.Section("ui").Key("contact_message").String(),
			"help_message":    app.config.Section("ui").Key("help_message").String(),
			"success_message": app.config.Section("ui").Key("success_message").String(),
		},
		"messages": {
			"message": app.config.Section("messages").Key("message").String(),
		},
	}
	msg, err := json.Marshal(messages)
	if err != nil {
		respond(500, "Failed to fetch default values", gc)
		return
	}
	gc.HTML(200, "setup.html", gin.H{
		"cssVersion": cssVersion,
		"lang":       app.storage.lang.Setup[lang],
		"emailLang":  app.storage.lang.Email[emailLang],
		"language":   app.storage.lang.Setup[lang].JSON,
		"messages":   string(msg),
	})
}

type testReq struct {
	ServerType    string `json:"type"`
	Server        string `json:"server"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Proxy         bool   `json:"proxy"`
	ProxyProtocol string `json:"proxy_protocol,omitempty"`
	ProxyAddress  string `json:"proxy_address,omitempty"`
	ProxyUsername string `json:"proxy_user,omitempty"`
	ProxyPassword string `json:"proxy_password,omitempty"`
}

func (app *appContext) TestJF(gc *gin.Context) {
	var req testReq
	gc.BindJSON(&req)
	if !(strings.HasPrefix(req.Server, "http://") || strings.HasPrefix(req.Server, "https://")) {
		req.Server = "http://" + req.Server
	}
	serverType := mediabrowser.JellyfinServer
	if req.ServerType == "emby" {
		serverType = mediabrowser.EmbyServer
	}
	tempjf, _ := mediabrowser.NewServer(serverType, req.Server, "jfa-go-setup", app.version, "auth", "auth", mediabrowser.NewNamedTimeoutHandler("authJF", req.Server, true), 30)

	if req.Proxy {
		conf := easyproxy.ProxyConfig{
			Protocol: easyproxy.HTTP,
			Addr:     req.ProxyAddress,
			User:     req.ProxyUsername,
			Password: req.ProxyPassword,
		}
		if strings.Contains(req.ProxyProtocol, "socks") {
			conf.Protocol = easyproxy.SOCKS5
		}

		transport, err := easyproxy.NewTransport(conf)
		if err != nil {
			respond(400, "errorProxy", gc)
			return
		}
		tempjf.SetTransport(transport)
	}

	user, status, err := tempjf.Authenticate(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		msg := ""
		switch status {
		case 0:
			msg = "errorConnectionRefused"
			status = 500
		case 401:
			msg = "errorInvalidUserPass"
		case 403:
			msg = "errorUserDisabled"
		case 404:
			msg = "error404"
		}
		app.info.Printf("Auth failed with code %d (%s)", status, err)
		if msg != "" {
			respond(status, msg, gc)
		} else {
			respondBool(status, false, gc)
		}
		return
	}
	if !user.Policy.IsAdministrator {
		respond(403, "errorNotAdmin", gc)
		return
	}
	gc.JSON(200, map[string]bool{"success": true})
}

// The first filesystem passed should be the localFS, to ensure the local lang files are loaded first.
func (st *Storage) loadLangSetup(filesystems ...fs.FS) error {
	st.lang.Setup = map[string]setupLang{}
	var english setupLang
	loadedLangs := make([]map[string]bool, len(filesystems))
	var load loadLangFunc
	load = func(fsIndex int, fname string) error {
		filesystem := filesystems[fsIndex]
		index := strings.TrimSuffix(fname, filepath.Ext(fname))
		lang := setupLang{}
		f, err := fs.ReadFile(filesystem, FSJoin(st.lang.SetupPath, fname))
		if err != nil {
			return err
		}
		err = json.Unmarshal(f, &lang)
		if err != nil {
			return err
		}
		st.lang.Common.patchCommonStrings(&lang.Strings, index)
		st.lang.Common.patchCommonNotifications(&lang.Notifications, index)
		if fname != "en-us.json" {
			if lang.Meta.Fallback != "" {
				fallback, ok := st.lang.Setup[lang.Meta.Fallback]
				err = nil
				if !ok {
					err = load(fsIndex, lang.Meta.Fallback+".json")
					fallback = st.lang.Setup[lang.Meta.Fallback]
				}
				if err == nil {
					loadedLangs[fsIndex][lang.Meta.Fallback+".json"] = true
					patchLang(&lang.Strings, &fallback.Strings, &english.Strings)
					patchLang(&lang.StartPage, &fallback.StartPage, &english.StartPage)
					patchLang(&lang.Updates, &fallback.Updates, &english.Updates)
					patchLang(&lang.Proxy, &fallback.Proxy, &english.Proxy)
					patchLang(&lang.EndPage, &fallback.EndPage, &english.EndPage)
					patchLang(&lang.Language, &fallback.Language, &english.Language)
					patchLang(&lang.Login, &fallback.Login, &english.Login)
					patchLang(&lang.JellyfinEmby, &fallback.JellyfinEmby, &english.JellyfinEmby)
					patchLang(&lang.Email, &fallback.Email, &english.Email)
					patchLang(&lang.Messages, &fallback.Messages, &english.Messages)
					patchLang(&lang.Notifications, &fallback.Notifications, &english.Notifications)
					patchLang(&lang.UserPage, &fallback.UserPage, &english.UserPage)
					patchLang(&lang.PasswordResets, &fallback.PasswordResets, &english.PasswordResets)
					patchLang(&lang.InviteEmails, &fallback.InviteEmails, &english.InviteEmails)
					patchLang(&lang.PasswordValidation, &fallback.PasswordValidation, &english.PasswordValidation)
					patchLang(&lang.HelpMessages, &fallback.HelpMessages, &english.HelpMessages)
				}
			}
			if (lang.Meta.Fallback != "" && err != nil) || lang.Meta.Fallback == "" {
				patchLang(&lang.Strings, &english.Strings)
				patchLang(&lang.StartPage, &english.StartPage)
				patchLang(&lang.Updates, &english.Updates)
				patchLang(&lang.Proxy, &english.Proxy)
				patchLang(&lang.EndPage, &english.EndPage)
				patchLang(&lang.Language, &english.Language)
				patchLang(&lang.Login, &english.Login)
				patchLang(&lang.JellyfinEmby, &english.JellyfinEmby)
				patchLang(&lang.Email, &english.Email)
				patchLang(&lang.Messages, &english.Messages)
				patchLang(&lang.Notifications, &english.Notifications)
				patchLang(&lang.UserPage, &english.UserPage)
				patchLang(&lang.PasswordResets, &english.PasswordResets)
				patchLang(&lang.InviteEmails, &english.InviteEmails)
				patchLang(&lang.PasswordValidation, &english.PasswordValidation)
				patchLang(&lang.HelpMessages, &english.HelpMessages)
			}
		}
		stringSettings, err := json.Marshal(lang)
		if err != nil {
			return err
		}
		lang.JSON = string(stringSettings)
		st.lang.Setup[index] = lang
		return nil
	}
	engFound := false
	var err error
	for i := range filesystems {
		loadedLangs[i] = map[string]bool{}
		err = load(i, "en-us.json")
		if err == nil {
			engFound = true
		}
		loadedLangs[i]["en-us.json"] = true
	}
	if !engFound {
		return err
	}
	english = st.lang.Setup["en-us"]
	setupLoaded := false
	for i := range filesystems {
		files, err := fs.ReadDir(filesystems[i], st.lang.SetupPath)
		if err != nil {
			return err
		}
		for _, f := range files {
			if !loadedLangs[i][f.Name()] {
				err = load(i, f.Name())
				if err == nil {
					setupLoaded = true
					loadedLangs[i][f.Name()] = true
				}
			}
		}
	}
	if !setupLoaded {
		return err
	}
	return nil
}
