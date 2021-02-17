package main

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/common"
	"github.com/hrfee/jfa-go/mediabrowser"
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
		"email": {
			"message": app.config.Section("email").Key("message").String(),
		},
	}
	msg, err := json.Marshal(messages)
	if err != nil {
		respond(500, "Failed to fetch default values", gc)
		return
	}
	gc.HTML(200, "setup.html", gin.H{
		"lang":      app.storage.lang.Setup[lang],
		"emailLang": app.storage.lang.Email[emailLang],
		"language":  app.storage.lang.Setup[lang].JSON,
		"messages":  string(msg),
	})
}

type testReq struct {
	ServerType string `json:"type"`
	Server     string `json:"server"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

func (app *appContext) TestJF(gc *gin.Context) {
	var req testReq
	gc.BindJSON(&req)
	serverType := mediabrowser.JellyfinServer
	if req.ServerType == "emby" {
		serverType = mediabrowser.EmbyServer
	}
	tempjf, _ := mediabrowser.NewServer(serverType, req.Server, "jfa-go-setup", app.version, "auth", "auth", common.NewTimeoutHandler("authJF", req.Server, true), 30)
	_, status, err := tempjf.Authenticate(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		app.info.Printf("Auth failed with code %d (%s)", status, err)
		gc.JSON(401, map[string]bool{"success": false})
		return
	}
	gc.JSON(200, map[string]bool{"success": true})
}

// The first filesystem passed should be the localFS, to ensure the local lang files are loaded first.
func (st *Storage) loadLangSetup(filesystems ...fs.FS) error {
	st.lang.Setup = map[string]setupLang{}
	var english setupLang
	load := func(filesystem fs.FS, fname string) error {
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
		st.lang.Common.patchCommon(index, &lang.Strings)
		if fname != "en-us.json" {
			patchLang(&english.Strings, &lang.Strings)
			patchLang(&english.StartPage, &lang.StartPage)
			patchLang(&english.EndPage, &lang.EndPage)
			patchLang(&english.Language, &lang.Language)
			patchLang(&english.Login, &lang.Login)
			patchLang(&english.JellyfinEmby, &lang.JellyfinEmby)
			patchLang(&english.Email, &lang.Email)
			patchLang(&english.Notifications, &lang.Notifications)
			patchLang(&english.PasswordResets, &lang.PasswordResets)
			patchLang(&english.InviteEmails, &lang.InviteEmails)
			patchLang(&english.PasswordValidation, &lang.PasswordValidation)
			patchLang(&english.HelpMessages, &lang.HelpMessages)
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
	for _, filesystem := range filesystems {
		err = load(filesystem, "en-us.json")
		if err == nil {
			engFound = true
		}
	}
	if !engFound {
		return err
	}
	english = st.lang.Setup["en-us"]
	setupLoaded := false
	for _, filesystem := range filesystems {
		files, err := fs.ReadDir(filesystem, st.lang.SetupPath)
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.Name() != "en-us.json" {
				err = load(filesystem, f.Name())
				if err == nil {
					setupLoaded = true
				}
			}
		}
	}
	if !setupLoaded {
		return err
	}
	return nil
}
