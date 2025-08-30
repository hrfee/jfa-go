package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/common"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/hrfee/mediabrowser"
	"github.com/itchyny/timefmt-go"
	"github.com/lithammer/shortuuid/v3"
	"gopkg.in/ini.v1"
)

func respond(code int, message string, gc *gin.Context) {
	resp := stringResponse{}
	if code == 200 || code == 204 {
		resp.Response = message
	} else {
		resp.Error = message
	}
	gc.JSON(code, resp)
	gc.Abort()
}

func respondBool(code int, val bool, gc *gin.Context) {
	resp := boolResponse{}
	if !val {
		resp.Error = true
	} else {
		resp.Success = true
	}
	gc.JSON(code, resp)
	gc.Abort()
}

func prettyTime(dt time.Time) (date, time string) {
	date = timefmt.Format(dt, datePattern)
	time = timefmt.Format(dt, timePattern)
	return
}

func formatDatetime(dt time.Time) string {
	d, t := prettyTime(dt)
	return d + " " + t
}

// https://stackoverflow.com/questions/36530251/time-since-with-months-and-years/36531443#36531443 THANKS
func timeDiff(a, b time.Time) (year, month, day, hour, min, sec int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = int(y2 - y1)
	month = int(M2 - M1)
	day = int(d2 - d1)
	hour = int(h2 - h1)
	min = int(m2 - m1)
	sec = int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}
	return
}

// Routes from now on!

// @Summary Resets a user's password with a PIN, and optionally set a new password if given.
// @Produce json
// @Success 200 {object} boolResponse
// @Success 400 {object} PasswordValidation
// @Failure 500 {object} boolResponse
// @Param ResetPasswordDTO body ResetPasswordDTO true "Pin and optional Password."
// @Router /reset [post]
// @tags Other
func (app *appContext) ResetSetPassword(gc *gin.Context) {
	var req ResetPasswordDTO
	gc.BindJSON(&req)
	validation := app.validator.validate(req.Password)
	captcha := app.config.Section("captcha").Key("enabled").MustBool(false)
	valid := true
	for _, val := range validation {
		if !val {
			valid = false
		}
	}
	if !valid || req.PIN == "" {
		app.info.Printf(lm.FailedChangePassword, lm.Jellyfin, "?", lm.InvalidPassword)
		gc.JSON(400, validation)
		return
	}
	isInternal := false

	if captcha && !app.verifyCaptcha(req.PIN, req.PIN, req.CaptchaText, true) {
		app.info.Printf(lm.FailedChangePassword, lm.Jellyfin, "?", lm.IncorrectCaptcha)
		respond(400, "errorCaptcha", gc)
		return
	}

	var userID, username string
	if reset, ok := app.internalPWRs[req.PIN]; ok {
		isInternal = true
		if time.Now().After(reset.Expiry) {
			app.info.Printf(lm.FailedChangePassword, lm.Jellyfin, "?", fmt.Sprintf(lm.ExpiredPIN, reset.PIN))
			respondBool(401, false, gc)
			delete(app.internalPWRs, req.PIN)
			return
		}
		userID = reset.ID
		username = reset.Username

		err := app.jf.ResetPasswordAdmin(userID)
		if err != nil {
			app.err.Printf(lm.FailedChangePassword, lm.Jellyfin, userID, err)
			respondBool(500, false, gc)
			return
		}
		delete(app.internalPWRs, req.PIN)
	} else {
		resp, err := app.jf.ResetPassword(req.PIN)
		if err != nil || !resp.Success {
			app.err.Printf(lm.FailedChangePassword, lm.Jellyfin, userID, err)
			respondBool(500, false, gc)
			return
		}
		if req.Password == "" || len(resp.UsersReset) == 0 {
			respondBool(200, false, gc)
			return
		}
		username = resp.UsersReset[0]
	}

	var user mediabrowser.User
	var err error
	if isInternal {
		user, err = app.jf.UserByID(userID, false)
	} else {
		user, err = app.jf.UserByName(username, false)
	}
	if err != nil {
		app.err.Printf(lm.FailedGetUser, userID, lm.Jellyfin, err)
		respondBool(500, false, gc)
		return
	}

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityResetPassword,
		UserID:     user.ID,
		SourceType: ActivityUser,
		Source:     user.ID,
		Time:       time.Now(),
	}, gc, true)

	prevPassword := req.PIN
	if isInternal {
		prevPassword = ""
	}
	err = app.jf.SetPassword(user.ID, prevPassword, req.Password)
	if err != nil {
		app.err.Printf(lm.FailedChangePassword, lm.Jellyfin, user.ID, err)
		respondBool(500, false, gc)
		return
	}
	if app.config.Section("ombi").Key("enabled").MustBool(false) {
		// This makes no sense so has been commented out.
		// It probably did at some point in the past.
		/* Silently fail for changing ombi passwords
		if (status != 200 && status != 204) || err != nil {
			app.err.Printf(lm.FailedGetUser, user.ID, lm.Jellyfin, err)
			respondBool(200, true, gc)
			return
		} */
		ombiUser, err := app.getOmbiUser(user.ID)
		if err != nil {
			app.err.Printf(lm.FailedGetUser, user.ID, lm.Ombi, err)
			respondBool(200, true, gc)
			return
		}
		ombiUser["password"] = req.Password
		err = app.ombi.ModifyUser(ombiUser)
		if err != nil {
			app.err.Printf(lm.FailedChangePassword, lm.Ombi, user.ID, err)
			respondBool(200, true, gc)
			return
		}
		app.debug.Printf(lm.ChangePassword, lm.Ombi, user.ID)
	}
	respondBool(200, true, gc)
}

// @Summary Get jfa-go configuration.
// @Produce json
// @Success 200 {object} common.Config "Uses the same format as config-base.json"
// @Router /config [get]
// @Security Bearer
// @tags Configuration
func (app *appContext) GetConfig(gc *gin.Context) {
	if discordEnabled {
		app.PatchConfigDiscordRoles()
	}
	gc.JSON(200, app.patchedConfig)
}

// @Summary Modify app config.
// @Produce json
// @Param appConfig body configDTO true "Config split into sections as in config.ini, all values as strings (lists split with | delimiter)."
// @Success 200 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /config [post]
// @Security Bearer
// @tags Configuration
func (app *appContext) ModifyConfig(gc *gin.Context) {
	var req configDTO
	gc.BindJSON(&req)
	// Load a new config, as we set various default values in app.config that shouldn't be stored.
	tempConfig, _ := ini.ShadowLoad(app.configPath)
	for _, section := range app.configBase.Sections {
		ns, ok := req[section.Section]
		if !ok {
			continue
		}
		newSection := ns.(map[string]any)
		iniSection, err := tempConfig.GetSection(section.Section)
		if err != nil {
			iniSection, err = tempConfig.NewSection(section.Section)
			if err != nil {
				app.err.Printf(lm.FailedModifyConfig, app.configPath, err)
				respond(500, err.Error(), gc)
				return
			}
		}
		for _, setting := range section.Settings {
			newValue, ok := newSection[setting.Setting]
			if !ok {
				continue
			}
			// Patch disabled to actually be an empty string
			if section.Section == "email" && setting.Setting == "method" && newValue == "disabled" {
				newValue = ""
			}
			// Copy language preference for chatbots to root one in "telegram"
			if (section.Section == "discord" || section.Section == "matrix") && setting.Setting == "language" {
				iniSection.Key("language").SetValue(newValue.(string))
			} else if setting.Type == common.ListType {
				splitValues := strings.Split(newValue.(string), "|")
				// Delete the key first to get rid of any shadow values
				iniSection.DeleteKey(setting.Setting)
				for i, v := range splitValues {
					if i == 0 {
						iniSection.Key(setting.Setting).SetValue(v)
					} else {
						iniSection.Key(setting.Setting).AddShadow(v)
					}
				}

			} else if newValue.(string) != iniSection.Key(setting.Setting).MustString("") {
				iniSection.Key(setting.Setting).SetValue(newValue.(string))
			}
		}
	}

	tempConfig.Section("").Key("first_run").SetValue("false")
	if err := tempConfig.SaveTo(app.configPath); err != nil {
		app.err.Printf(lm.FailedWriting, app.configPath, err)
		respond(500, err.Error(), gc)
		return
	}
	app.info.Printf(lm.ModifyConfig, app.configPath)
	gc.JSON(200, map[string]bool{"success": true})
	if req["restart-program"] != nil && req["restart-program"].(bool) {
		app.Restart()
	}
	app.ReloadConfig()
	// Patch new settings for next GetConfig
	app.PatchConfigBase()
	// Reinitialize password validator on config change, as opposed to every applicable request like in python.
	if _, ok := req["password_validation"]; ok {
		validatorConf := ValidatorConf{
			"length":    app.config.Section("password_validation").Key("min_length").MustInt(0),
			"uppercase": app.config.Section("password_validation").Key("upper").MustInt(0),
			"lowercase": app.config.Section("password_validation").Key("lower").MustInt(0),
			"number":    app.config.Section("password_validation").Key("number").MustInt(0),
			"special":   app.config.Section("password_validation").Key("special").MustInt(0),
		}
		if !app.config.Section("password_validation").Key("enabled").MustBool(false) {
			for key := range validatorConf {
				validatorConf[key] = 0
			}
		}
		app.validator.init(validatorConf)
	}
}

// @Summary Returns whether there's a new update, and extra info if there is.
// @Produce json
// @Success 200 {object} checkUpdateDTO
// @Router /config/update [get]
// @Security Bearer
// @tags Configuration
func (app *appContext) CheckUpdate(gc *gin.Context) {
	if !app.newUpdate {
		app.update = Update{}
	}
	gc.JSON(200, checkUpdateDTO{New: app.newUpdate, Update: app.update})
}

// @Summary Apply an update.
// @Produce json
// @Success 200 {object} boolResponse
// @Success 400 {object} stringResponse
// @Success 500 {object} boolResponse
// @Router /config/update [post]
// @Security Bearer
// @tags Configuration
func (app *appContext) ApplyUpdate(gc *gin.Context) {
	if !app.update.CanUpdate {
		app.info.Printf(lm.FailedApplyUpdate, lm.UpdateManual)
		respond(400, lm.UpdateManual, gc)
		return
	}
	err := app.update.update()
	if err != nil {
		app.err.Printf(lm.FailedApplyUpdate, err)
		respondBool(500, false, gc)
		return
	}
	if PLATFORM == "windows" {
		respondBool(500, true, gc)
		return
	}
	respondBool(200, true, gc)
	app.HardRestart()
}

// @Summary Logout by deleting refresh token from cookies.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /logout [post]
// @Security Bearer
// @tags Other
func (app *appContext) Logout(gc *gin.Context) {
	cookie, err := gc.Cookie("refresh")
	if err != nil {
		msg := fmt.Sprintf(lm.FailedGetCookies, "refresh", err)
		app.debug.Println(msg)
		respond(500, msg, gc)
		return
	}
	app.invalidTokens = append(app.invalidTokens, cookie)
	gc.SetCookie("refresh", "invalid", -1, "/", gc.Request.URL.Hostname(), true, true)
	respondBool(200, true, gc)
}

// @Summary Returns a map of available language codes to their full names, usable in the lang query parameter.
// @Produce json
// @Success 200 {object} langDTO
// @Failure 500 {object} stringResponse
// @Param page path string true "admin/form/setup/email/pwr"
// @Router /lang/{page} [get]
// @tags Other
func (app *appContext) GetLanguages(gc *gin.Context) {
	page := gc.Param("page")
	resp := langDTO{}
	switch page {
	case "form", "user":
		for key, lang := range app.storage.lang.User {
			resp[key] = lang.Meta.Name
		}
	case "admin":
		for key, lang := range app.storage.lang.Admin {
			resp[key] = lang.Meta.Name
		}
	case "setup":
		for key, lang := range app.storage.lang.Setup {
			resp[key] = lang.Meta.Name
		}
	case "email":
		for key, lang := range app.storage.lang.Email {
			resp[key] = lang.Meta.Name
		}
	case "pwr":
		for key, lang := range app.storage.lang.PasswordReset {
			resp[key] = lang.Meta.Name
		}
	}
	if len(resp) == 0 {
		respond(500, "Couldn't get languages", gc)
		return
	}
	gc.JSON(200, resp)
}

// @Summary Serves a translations for pages "admin" or "form".
// @Produce json
// @Success 200 {object} adminLang
// @Failure 400 {object} boolResponse
// @Param page path string true "admin or form."
// @Param language path string true "language code, e.g en-us."
// @Router /lang/{page}/{language} [get]
// @tags Other
func (app *appContext) ServeLang(gc *gin.Context) {
	page := gc.Param("page")
	lang := strings.Replace(gc.Param("file"), ".json", "", 1)
	if page == "admin" {
		gc.JSON(200, app.storage.lang.Admin[lang])
		return
	} else if page == "form" || page == "user" {
		gc.JSON(200, app.storage.lang.User[lang])
		return
	}
	respondBool(400, false, gc)
}

// @Summary Restarts the program. No response means success.
// @Router /restart [post]
// @Security Bearer
// @tags Other
func (app *appContext) restart(gc *gin.Context) {
	app.Restart()
}

// @Summary Returns the last 100 lines of the log.
// @Router /log [get]
// @Success 200 {object} LogDTO
// @Security Bearer
// @tags Other
func (app *appContext) GetLog(gc *gin.Context) {
	gc.JSON(200, LogDTO{lineCache.String()})
}

// no need to syscall.exec anymore!
func (app *appContext) Restart() error {
	app.info.Println(lm.Restarting)
	if TRAY {
		TRAYRESTART <- true
	} else {
		RESTART <- true
	}
	// Safety Sleep (Ensure shutdown tasks get done)
	time.Sleep(time.Second)
	return nil
}
