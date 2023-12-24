package main

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

func (app *appContext) loadStrftime() {
	app.datePattern = app.config.Section("messages").Key("date_format").String()
	app.timePattern = `%H:%M`
	if val, _ := app.config.Section("messages").Key("use_24h").Bool(); !val {
		app.timePattern = `%I:%M %p`
	}
	return
}

func (app *appContext) prettyTime(dt time.Time) (date, time string) {
	date = timefmt.Format(dt, app.datePattern)
	time = timefmt.Format(dt, app.timePattern)
	return
}

func (app *appContext) formatDatetime(dt time.Time) string {
	d, t := app.prettyTime(dt)
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
		app.info.Printf("%s: Password reset failed: Invalid password", req.PIN)
		gc.JSON(400, validation)
		return
	}
	isInternal := false

	if captcha && !app.verifyCaptcha(req.PIN, req.PIN, req.CaptchaText, true) {
		app.info.Printf("%s: PWR Failed: Captcha Incorrect", req.PIN)
		respond(400, "errorCaptcha", gc)
		return
	}

	var userID, username string
	if reset, ok := app.internalPWRs[req.PIN]; ok {
		isInternal = true
		if time.Now().After(reset.Expiry) {
			app.info.Printf("Password reset failed: PIN \"%s\" has expired", reset.PIN)
			respondBool(401, false, gc)
			delete(app.internalPWRs, req.PIN)
			return
		}
		userID = reset.ID
		username = reset.Username

		status, err := app.jf.ResetPasswordAdmin(userID)
		if !(status == 200 || status == 204) || err != nil {
			app.err.Printf("Password Reset failed (%d): %v", status, err)
			respondBool(status, false, gc)
			return
		}
		delete(app.internalPWRs, req.PIN)
	} else {
		resp, status, err := app.jf.ResetPassword(req.PIN)
		if status != 200 || err != nil || !resp.Success {
			app.err.Printf("Password Reset failed (%d): %v", status, err)
			respondBool(status, false, gc)
			return
		}
		if req.Password == "" || len(resp.UsersReset) == 0 {
			respondBool(200, false, gc)
			return
		}
		username = resp.UsersReset[0]
	}

	var user mediabrowser.User
	var status int
	var err error
	if isInternal {
		user, status, err = app.jf.UserByID(userID, false)
	} else {
		user, status, err = app.jf.UserByName(username, false)
	}
	if status != 200 || err != nil {
		app.err.Printf("Failed to get user \"%s\" (%d): %v", username, status, err)
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
	status, err = app.jf.SetPassword(user.ID, prevPassword, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("Failed to change password for \"%s\" (%d): %v", username, status, err)
		respondBool(500, false, gc)
		return
	}
	if app.config.Section("ombi").Key("enabled").MustBool(false) {
		// Silently fail for changing ombi passwords
		if (status != 200 && status != 204) || err != nil {
			app.err.Printf("Failed to get user \"%s\" from jellyfin/emby (%d): %v", username, status, err)
			respondBool(200, true, gc)
			return
		}
		ombiUser, status, err := app.getOmbiUser(user.ID)
		if status != 200 || err != nil {
			app.err.Printf("Failed to get user \"%s\" from ombi (%d): %v", username, status, err)
			respondBool(200, true, gc)
			return
		}
		ombiUser["password"] = req.Password
		status, err = app.ombi.ModifyUser(ombiUser)
		if status != 200 || err != nil {
			app.err.Printf("Failed to set password for ombi user \"%s\" (%d): %v", ombiUser["userName"], status, err)
			respondBool(200, true, gc)
			return
		}
		app.debug.Printf("Reset password for ombi user \"%s\"", ombiUser["userName"])
	}
	respondBool(200, true, gc)
}

// @Summary Get jfa-go configuration.
// @Produce json
// @Success 200 {object} settings "Uses the same format as config-base.json"
// @Router /config [get]
// @Security Bearer
// @tags Configuration
func (app *appContext) GetConfig(gc *gin.Context) {
	app.info.Println("Config requested")
	resp := app.configBase
	// Load language options
	formOptions := app.storage.lang.User.getOptions()
	fl := resp.Sections["ui"].Settings["language-form"]
	fl.Options = formOptions
	fl.Value = app.config.Section("ui").Key("language-form").MustString("en-us")
	pwrOptions := app.storage.lang.PasswordReset.getOptions()
	pl := resp.Sections["password_resets"].Settings["language"]
	pl.Options = pwrOptions
	pl.Value = app.config.Section("password_resets").Key("language").MustString("en-us")
	adminOptions := app.storage.lang.Admin.getOptions()
	al := resp.Sections["ui"].Settings["language-admin"]
	al.Options = adminOptions
	al.Value = app.config.Section("ui").Key("language-admin").MustString("en-us")
	emailOptions := app.storage.lang.Email.getOptions()
	el := resp.Sections["email"].Settings["language"]
	el.Options = emailOptions
	el.Value = app.config.Section("email").Key("language").MustString("en-us")
	telegramOptions := app.storage.lang.Email.getOptions()
	tl := resp.Sections["telegram"].Settings["language"]
	tl.Options = telegramOptions
	tl.Value = app.config.Section("telegram").Key("language").MustString("en-us")
	if updater == "" {
		delete(resp.Sections, "updates")
		for i, v := range resp.Order {
			if v == "updates" {
				resp.Order = append(resp.Order[:i], resp.Order[i+1:]...)
				break
			}
		}
	}
	if PLATFORM == "windows" {
		delete(resp.Sections["smtp"].Settings, "ssl_cert")
		for i, v := range resp.Sections["smtp"].Order {
			if v == "ssl_cert" {
				sect := resp.Sections["smtp"]
				sect.Order = append(sect.Order[:i], sect.Order[i+1:]...)
				resp.Sections["smtp"] = sect
			}
		}
	}
	if !MatrixE2EE() {
		delete(resp.Sections["matrix"].Settings, "encryption")
		for i, v := range resp.Sections["matrix"].Order {
			if v == "encryption" {
				sect := resp.Sections["matrix"]
				sect.Order = append(sect.Order[:i], sect.Order[i+1:]...)
				resp.Sections["matrix"] = sect
			}
		}
	}
	for sectName, section := range resp.Sections {
		for settingName, setting := range section.Settings {
			val := app.config.Section(sectName).Key(settingName)
			s := resp.Sections[sectName].Settings[settingName]
			switch setting.Type {
			case "text", "email", "select", "password", "note":
				s.Value = val.MustString("")
			case "number":
				s.Value = val.MustInt(0)
			case "bool":
				s.Value = val.MustBool(false)
			}
			resp.Sections[sectName].Settings[settingName] = s
		}
	}
	if discordEnabled {
		r, err := app.discord.ListRoles()
		if err == nil {
			roles := make([][2]string, len(r)+1)
			roles[0] = [2]string{"", "None"}
			for i, role := range r {
				roles[i+1] = role
			}
			s := resp.Sections["discord"].Settings["apply_role"]
			s.Options = roles
			resp.Sections["discord"].Settings["apply_role"] = s
		}
	}

	resp.Sections["ui"].Settings["language-form"] = fl
	resp.Sections["ui"].Settings["language-admin"] = al
	resp.Sections["email"].Settings["language"] = el
	resp.Sections["password_resets"].Settings["language"] = pl
	resp.Sections["telegram"].Settings["language"] = tl
	resp.Sections["discord"].Settings["language"] = tl
	resp.Sections["matrix"].Settings["language"] = tl

	// if setting := resp.Sections["invite_emails"].Settings["url_base"]; setting.Value == "" {
	// 	setting.Value = strings.TrimSuffix(resp.Sections["password_resets"].Settings["url_base"].Value.(string), "/invite")
	// 	resp.Sections["invite_emails"].Settings["url_base"] = setting
	// }
	// if setting := resp.Sections["password_resets"].Settings["url_base"]; setting.Value == "" {
	// 	setting.Value = strings.TrimSuffix(resp.Sections["invite_emails"].Settings["url_base"].Value.(string), "/invite")
	// 	resp.Sections["password_resets"].Settings["url_base"] = setting
	// }

	gc.JSON(200, resp)
}

// @Summary Modify app config.
// @Produce json
// @Param appConfig body configDTO true "Config split into sections as in config.ini, all values as strings."
// @Success 200 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /config [post]
// @Security Bearer
// @tags Configuration
func (app *appContext) ModifyConfig(gc *gin.Context) {
	app.info.Println("Config modification requested")
	var req configDTO
	gc.BindJSON(&req)
	// Load a new config, as we set various default values in app.config that shouldn't be stored.
	tempConfig, _ := ini.Load(app.configPath)
	for section, settings := range req {
		if section != "restart-program" {
			_, err := tempConfig.GetSection(section)
			if err != nil {
				tempConfig.NewSection(section)
			}
			for setting, value := range settings.(map[string]interface{}) {
				if section == "email" && setting == "method" && value == "disabled" {
					value = ""
				}
				if (section == "discord" || section == "matrix") && setting == "language" {
					tempConfig.Section("telegram").Key("language").SetValue(value.(string))
				} else if value.(string) != app.config.Section(section).Key(setting).MustString("") {
					tempConfig.Section(section).Key(setting).SetValue(value.(string))
				}
			}
		}
	}
	tempConfig.Section("").Key("first_run").SetValue("false")
	if err := tempConfig.SaveTo(app.configPath); err != nil {
		app.err.Printf("Failed to save config to \"%s\": %v", app.configPath, err)
		respond(500, err.Error(), gc)
		return
	}
	app.debug.Println("Config saved")
	gc.JSON(200, map[string]bool{"success": true})
	if req["restart-program"] != nil && req["restart-program"].(bool) {
		app.info.Println("Restarting...")
		if TRAY {
			TRAYRESTART <- true
		} else {
			RESTART <- true
		}
		// Safety Sleep (Ensure shutdown tasks get done)
		time.Sleep(time.Second)
	}
	app.loadConfig()
	// Reinitialize password validator on config change, as opposed to every applicable request like in python.
	if _, ok := req["password_validation"]; ok {
		app.debug.Println("Reinitializing validator")
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
		respond(400, "Update is manual", gc)
		return
	}
	err := app.update.update()
	if err != nil {
		app.err.Printf("Failed to apply update: %v", err)
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
		app.debug.Printf("Couldn't get cookies: %s", err)
		respond(500, "Couldn't fetch cookies", gc)
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
	app.info.Println("Restarting...")
	err := app.Restart()
	if err != nil {
		app.err.Printf("Couldn't restart, try restarting manually: %v", err)
	}
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
	if TRAY {
		TRAYRESTART <- true
	} else {
		RESTART <- true
	}
	// Safety Sleep (Ensure shutdown tasks get done)
	time.Sleep(time.Second)
	return nil
}
