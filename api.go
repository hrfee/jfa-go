package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/knz/strtime"
	"github.com/lithammer/shortuuid/v3"
	"gopkg.in/ini.v1"
)

func (app *appContext) loadStrftime() {
	app.datePattern = app.config.Section("email").Key("date_format").String()
	app.timePattern = `%H:%M`
	if val, _ := app.config.Section("email").Key("use_24h").Bool(); !val {
		app.timePattern = `%I:%M %p`
	}
	return
}

func (app *appContext) prettyTime(dt time.Time) (date, time string) {
	date, _ = strtime.Strftime(dt, app.datePattern)
	time, _ = strtime.Strftime(dt, app.timePattern)
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

func (app *appContext) checkInvites() {
	current_time := time.Now()
	app.storage.loadInvites()
	changed := false
	for code, data := range app.storage.invites {
		expiry := data.ValidTill
		if current_time.After(expiry) {
			app.debug.Printf("Housekeeping: Deleting old invite %s", code)
			notify := data.Notify
			if app.config.Section("notifications").Key("enabled").MustBool(false) && len(notify) != 0 {
				app.debug.Printf("%s: Expiry notification", code)
				for address, settings := range notify {
					if settings["notify-expiry"] {
						go func() {
							if err := app.email.constructExpiry(code, data, app); err != nil {
								app.err.Printf("%s: Failed to construct expiry notification", code)
								app.debug.Printf("Error: %s", err)
							} else if err := app.email.send(address); err != nil {
								app.err.Printf("%s: Failed to send expiry notification", code)
								app.debug.Printf("Error: %s", err)
							} else {
								app.info.Printf("Sent expiry notification to %s", address)
							}
						}()
					}
				}
			}
			changed = true
			delete(app.storage.invites, code)
		}
	}
	if changed {
		app.storage.storeInvites()
	}
}

func (app *appContext) checkInvite(code string, used bool, username string) bool {
	current_time := time.Now()
	app.storage.loadInvites()
	changed := false
	if inv, match := app.storage.invites[code]; match {
		expiry := inv.ValidTill
		if current_time.After(expiry) {
			app.debug.Printf("Housekeeping: Deleting old invite %s", code)
			notify := inv.Notify
			if app.config.Section("notifications").Key("enabled").MustBool(false) && len(notify) != 0 {
				app.debug.Printf("%s: Expiry notification", code)
				for address, settings := range notify {
					if settings["notify-expiry"] {
						go func() {
							if err := app.email.constructExpiry(code, inv, app); err != nil {
								app.err.Printf("%s: Failed to construct expiry notification", code)
								app.debug.Printf("Error: %s", err)
							} else if err := app.email.send(address); err != nil {
								app.err.Printf("%s: Failed to send expiry notification", code)
								app.debug.Printf("Error: %s", err)
							} else {
								app.info.Printf("Sent expiry notification to %s", address)
							}
						}()
					}
				}
			}
			changed = true
			match = false
			delete(app.storage.invites, code)
		} else if used {
			changed = true
			del := false
			newInv := inv
			if newInv.RemainingUses == 1 {
				del = true
				delete(app.storage.invites, code)
			} else if newInv.RemainingUses != 0 {
				// 0 means infinite i guess?
				newInv.RemainingUses -= 1
			}
			newInv.UsedBy = append(newInv.UsedBy, []string{username, app.formatDatetime(current_time)})
			if !del {
				app.storage.invites[code] = newInv
			}
		}
		if changed {
			app.storage.storeInvites()
		}
		return match
	}
	return false
}

// Routes from now on!

type newUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Code     string `json:"code"`
}

func (app *appContext) NewUser(gc *gin.Context) {
	var req newUserReq
	gc.BindJSON(&req)
	app.debug.Printf("%s: New user attempt", req.Code)
	if !app.checkInvite(req.Code, false, "") {
		app.info.Printf("%s New user failed: invalid code", req.Code)
		gc.JSON(401, map[string]bool{"success": false})
		gc.Abort()
		return
	}
	validation := app.validator.validate(req.Password)
	valid := true
	for _, val := range validation {
		if !val {
			valid = false
		}
	}
	if !valid {
		// 200 bcs idk what i did in js
		app.info.Printf("%s New user failed: Invalid password", req.Code)
		gc.JSON(200, validation)
		gc.Abort()
		return
	}
	existingUser, _, _ := app.jf.userByName(req.Username, false)
	if existingUser != nil {
		msg := fmt.Sprintf("User already exists named %s", req.Username)
		app.info.Printf("%s New user failed: %s", req.Code, msg)
		respond(401, msg, gc)
		return
	}
	user, status, err := app.jf.newUser(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("%s New user failed: Jellyfin responded with %d", req.Code, status)
		respond(401, "Unknown error", gc)
		return
	}
	app.checkInvite(req.Code, true, req.Username)
	invite := app.storage.invites[req.Code]
	if app.config.Section("notifications").Key("enabled").MustBool(false) {
		for address, settings := range invite.Notify {
			if settings["notify-creation"] {
				go func() {
					if err := app.email.constructCreated(req.Code, req.Username, req.Email, invite, app); err != nil {
						app.err.Printf("%s: Failed to construct user creation notification", req.Code)
						app.debug.Printf("%s: Error: %s", req.Code, err)
					} else if err := app.email.send(address); err != nil {
						app.err.Printf("%s: Failed to send user creation notification", req.Code)
						app.debug.Printf("%s: Error: %s", req.Code, err)
					} else {
						app.info.Printf("%s: Sent user creation notification to %s", req.Code, address)
					}
				}()
			}
		}
	}
	var id string
	if user["Id"] != nil {
		id = user["Id"].(string)
	}
	if len(app.storage.policy) != 0 {
		status, err = app.jf.setPolicy(id, app.storage.policy)
		if !(status == 200 || status == 204) {
			app.err.Printf("%s: Failed to set user policy: Code %d", req.Code, status)
		}
	}
	if len(app.storage.configuration) != 0 && len(app.storage.displayprefs) != 0 {
		status, err = app.jf.setConfiguration(id, app.storage.configuration)
		if (status == 200 || status == 204) && err == nil {
			status, err = app.jf.setDisplayPreferences(id, app.storage.displayprefs)
		} else {
			app.err.Printf("%s: Failed to set configuration template: Code %d", req.Code, status)
		}
	}
	if app.config.Section("password_resets").Key("enabled").MustBool(false) {
		app.storage.emails[id] = req.Email
		app.storage.storeEmails()
	}
	if app.config.Section("ombi").Key("enabled").MustBool(false) {
		app.storage.loadOmbiTemplate()
		if len(app.storage.ombi_template) != 0 {
			errors, code, err := app.ombi.newUser(req.Username, req.Password, req.Email, app.storage.ombi_template)
			if err != nil || code != 200 {
				app.info.Printf("Failed to create Ombi user (%d): %s", code, err)
				app.debug.Printf("Errors reported by Ombi: %s", strings.Join(errors, ", "))
			} else {
				app.info.Println("Created Ombi user")
			}
		}
	}
	gc.JSON(200, validation)
}

type generateInviteReq struct {
	Days          int    `json:"days"`
	Hours         int    `json:"hours"`
	Minutes       int    `json:"minutes"`
	Email         string `json:"email"`
	MultipleUses  bool   `json:"multiple-uses"`
	NoLimit       bool   `json:"no-limit"`
	RemainingUses int    `json:"remaining-uses"`
}

func (app *appContext) GenerateInvite(gc *gin.Context) {
	var req generateInviteReq
	app.debug.Println("Generating new invite")
	app.storage.loadInvites()
	gc.BindJSON(&req)
	current_time := time.Now()
	valid_till := current_time.AddDate(0, 0, req.Days)
	valid_till = valid_till.Add(time.Hour*time.Duration(req.Hours) + time.Minute*time.Duration(req.Minutes))
	invite_code := shortuuid.New()
	var invite Invite
	invite.Created = current_time
	if req.MultipleUses {
		if req.NoLimit {
			invite.NoLimit = true
		} else {
			invite.RemainingUses = req.RemainingUses
		}
	} else {
		invite.RemainingUses = 1
	}
	invite.ValidTill = valid_till
	if req.Email != "" && app.config.Section("invite_emails").Key("enabled").MustBool(false) {
		app.debug.Printf("%s: Sending invite email", invite_code)
		invite.Email = req.Email
		if err := app.email.constructInvite(invite_code, invite, app); err != nil {
			invite.Email = fmt.Sprintf("Failed to send to %s", req.Email)
			app.err.Printf("%s: Failed to construct invite email", invite_code)
			app.debug.Printf("%s: Error: %s", invite_code, err)
		} else if err := app.email.send(req.Email); err != nil {
			invite.Email = fmt.Sprintf("Failed to send to %s", req.Email)
			app.err.Printf("%s: %s", invite_code, invite.Email)
			app.debug.Printf("%s: Error: %s", invite_code, err)
		} else {
			app.info.Printf("%s: Sent invite email to %s", invite_code, req.Email)
		}
	}
	app.storage.invites[invite_code] = invite
	app.storage.storeInvites()
	gc.JSON(200, map[string]bool{"success": true})
}

func (app *appContext) GetInvites(gc *gin.Context) {
	app.debug.Println("Invites requested")
	current_time := time.Now()
	app.storage.loadInvites()
	app.checkInvites()
	var invites []map[string]interface{}
	for code, inv := range app.storage.invites {
		_, _, days, hours, minutes, _ := timeDiff(inv.ValidTill, current_time)
		invite := make(map[string]interface{})
		invite["code"] = code
		invite["days"] = days
		invite["hours"] = hours
		invite["minutes"] = minutes
		invite["created"] = app.formatDatetime(inv.Created)
		if len(inv.UsedBy) != 0 {
			invite["used-by"] = inv.UsedBy
		}
		if inv.NoLimit {
			invite["no-limit"] = true
		}
		invite["remaining-uses"] = 1
		if inv.RemainingUses != 0 {
			invite["remaining-uses"] = inv.RemainingUses
		}
		if inv.Email != "" {
			invite["email"] = inv.Email
		}
		if len(inv.Notify) != 0 {
			var address string
			if app.config.Section("ui").Key("jellyfin_login").MustBool(false) {
				app.storage.loadEmails()
				address = app.storage.emails[gc.GetString("jfId")].(string)
			} else {
				address = app.config.Section("ui").Key("email").String()
			}
			if _, ok := inv.Notify[address]; ok {
				for _, notifyType := range []string{"notify-expiry", "notify-creation"} {
					if _, ok = inv.Notify[notifyType]; ok {
						invite[notifyType] = inv.Notify[address][notifyType]
					}
				}
			}
		}
		invites = append(invites, invite)
	}
	resp := map[string][]map[string]interface{}{
		"invites": invites,
	}
	gc.JSON(200, resp)
}

type notifySetting struct {
	NotifyExpiry   bool `json:"notify-expiry"`
	NotifyCreation bool `json:"notify-creation"`
}

func (app *appContext) SetNotify(gc *gin.Context) {
	var req map[string]notifySetting
	gc.BindJSON(&req)
	changed := false
	for code, settings := range req {
		app.debug.Printf("%s: Notification settings change requested", code)
		app.storage.loadInvites()
		app.storage.loadEmails()
		invite, ok := app.storage.invites[code]
		if !ok {
			app.err.Printf("%s Notification setting change failed: Invalid code", code)
			gc.JSON(400, map[string]string{"error": "Invalid invite code"})
			gc.Abort()
			return
		}
		var address string
		if app.config.Section("ui").Key("jellyfin_login").MustBool(false) {
			var ok bool
			address, ok = app.storage.emails[gc.GetString("jfId")].(string)
			if !ok {
				app.err.Printf("%s: Couldn't find email address. Make sure it's set", code)
				app.debug.Printf("%s: User ID \"%s\"", code, gc.GetString("jfId"))
				gc.JSON(500, map[string]string{"error": "Missing user email"})
				gc.Abort()
				return
			}
		} else {
			address = app.config.Section("ui").Key("email").String()
		}
		if invite.Notify == nil {
			invite.Notify = map[string]map[string]bool{}
		}
		if _, ok := invite.Notify[address]; !ok {
			invite.Notify[address] = map[string]bool{}
		} /*else {
		if _, ok := invite.Notify[address]["notify-expiry"]; !ok {
		*/
		if invite.Notify[address]["notify-expiry"] != settings.NotifyExpiry {
			invite.Notify[address]["notify-expiry"] = settings.NotifyExpiry
			app.debug.Printf("%s: Set \"notify-expiry\" to %t for %s", code, settings.NotifyExpiry, address)
			changed = true
		}
		if invite.Notify[address]["notify-creation"] != settings.NotifyCreation {
			invite.Notify[address]["notify-creation"] = settings.NotifyCreation
			app.debug.Printf("%s: Set \"notify-creation\" to %t for %s", code, settings.NotifyExpiry, address)
			changed = true
		}
		if changed {
			app.storage.invites[code] = invite
		}
	}
	if changed {
		app.storage.storeInvites()
	}
}

type deleteReq struct {
	Code string `json:"code"`
}

func (app *appContext) DeleteInvite(gc *gin.Context) {
	var req deleteReq
	gc.BindJSON(&req)
	app.debug.Printf("%s: Deletion requested", req.Code)
	var ok bool
	_, ok = app.storage.invites[req.Code]
	if ok {
		delete(app.storage.invites, req.Code)
		app.storage.storeInvites()
		app.info.Printf("%s: Invite deleted", req.Code)
		gc.JSON(200, map[string]bool{"success": true})
		return
	}
	app.err.Printf("%s: Deletion failed: Invalid code", req.Code)
	respond(401, "Code doesn't exist", gc)
}

type userResp struct {
	UserList []respUser `json:"users"`
}

type respUser struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

func (app *appContext) GetUsers(gc *gin.Context) {
	app.debug.Println("Users requested")
	var resp userResp
	resp.UserList = []respUser{}
	users, status, err := app.jf.getUsers(false)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("Failed to get users from Jellyfin: Code %d", status)
		app.debug.Printf("Error: %s", err)
		respond(500, "Couldn't get users", gc)
		return
	}
	for _, jfUser := range users {
		var user respUser
		user.Name = jfUser["Name"].(string)
		if email, ok := app.storage.emails[jfUser["Id"].(string)]; ok {
			user.Email = email.(string)
		}
		resp.UserList = append(resp.UserList, user)
	}
	gc.JSON(200, resp)
}

type ombiUser struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id"`
}

func (app *appContext) OmbiUsers(gc *gin.Context) {
	app.debug.Println("Ombi users requested")
	users, status, err := app.ombi.getUsers()
	if err != nil || status != 200 {
		app.err.Printf("Failed to get users from Ombi: Code %d", status)
		app.debug.Printf("Error: %s", err)
		respond(500, "Couldn't get users", gc)
		return
	}
	userlist := make([]ombiUser, len(users))
	for i, data := range users {
		userlist[i] = ombiUser{
			Name: data["userName"].(string),
			ID:   data["id"].(string),
		}
	}
	gc.JSON(200, map[string][]ombiUser{"users": userlist})
}

func (app *appContext) ModifyEmails(gc *gin.Context) {
	var req map[string]string
	gc.BindJSON(&req)
	app.debug.Println("Email modification requested")
	users, status, err := app.jf.getUsers(false)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("Failed to get users from Jellyfin: Code %d", status)
		app.debug.Printf("Error: %s", err)
		respond(500, "Couldn't get users", gc)
		return
	}
	for _, jfUser := range users {
		if address, ok := req[jfUser["Name"].(string)]; ok {
			app.storage.emails[jfUser["Id"].(string)] = address
		}
	}
	app.storage.storeEmails()
	app.info.Println("Email list modified")
	gc.JSON(200, map[string]bool{"success": true})
}

func (app *appContext) SetOmbiDefaults(gc *gin.Context) {
	var req ombiUser
	gc.BindJSON(&req)
	template, code, err := app.ombi.templateByID(req.ID)
	if err != nil || code != 200 || len(template) == 0 {
		app.err.Printf("Couldn't get user from Ombi: %d %s", code, err)
		respond(500, "Couldn't get user", gc)
		return
	}
	app.storage.ombi_template = template
	fmt.Println(app.storage.ombi_path)
	app.storage.storeOmbiTemplate()
	gc.JSON(200, map[string]bool{"success": true})
}

type defaultsReq struct {
	Username   string `json:"username"`
	Homescreen bool   `json:"homescreen"`
}

func (app *appContext) SetDefaults(gc *gin.Context) {
	var req defaultsReq
	gc.BindJSON(&req)
	app.info.Printf("Getting user defaults from \"%s\"", req.Username)
	user, status, err := app.jf.userByName(req.Username, false)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("Failed to get user from Jellyfin: Code %d", status)
		app.debug.Printf("Error: %s", err)
		respond(500, "Couldn't get user", gc)
		return
	}
	userID := user["Id"].(string)
	policy := user["Policy"].(map[string]interface{})
	app.storage.policy = policy
	app.storage.storePolicy()
	app.debug.Println("User policy template stored")
	if req.Homescreen {
		configuration := user["Configuration"].(map[string]interface{})
		var displayprefs map[string]interface{}
		displayprefs, status, err = app.jf.getDisplayPreferences(userID)
		if !(status == 200 || status == 204) || err != nil {
			app.err.Printf("Failed to get DisplayPrefs: Code %d", status)
			app.debug.Printf("Error: %s", err)
			respond(500, "Couldn't get displayprefs", gc)
			return
		}
		app.storage.configuration = configuration
		app.storage.displayprefs = displayprefs
		app.storage.storeConfiguration()
		app.debug.Println("Configuration template stored")
		app.storage.storeDisplayprefs()
		app.debug.Println("DisplayPrefs template stored")
	}
	gc.JSON(200, map[string]bool{"success": true})
}

func (app *appContext) GetConfig(gc *gin.Context) {
	app.info.Println("Config requested")
	resp := map[string]interface{}{}
	for section, settings := range app.configBase {
		if section == "order" {
			resp[section] = settings.([]interface{})
		} else {
			resp[section] = make(map[string]interface{})
			for key, values := range settings.(map[string]interface{}) {
				if key == "order" {
					resp[section].(map[string]interface{})[key] = values.([]interface{})
				} else {
					resp[section].(map[string]interface{})[key] = values.(map[string]interface{})
					if key != "meta" {
						dataType := resp[section].(map[string]interface{})[key].(map[string]interface{})["type"].(string)
						configKey := app.config.Section(section).Key(key)
						if dataType == "number" {
							if val, err := configKey.Int(); err == nil {
								resp[section].(map[string]interface{})[key].(map[string]interface{})["value"] = val
							}
						} else if dataType == "bool" {
							resp[section].(map[string]interface{})[key].(map[string]interface{})["value"] = configKey.MustBool(false)
						} else {
							resp[section].(map[string]interface{})[key].(map[string]interface{})["value"] = configKey.String()
						}
					}
				}
			}
		}
	}
	gc.JSON(200, resp)
}

func (app *appContext) ModifyConfig(gc *gin.Context) {
	app.info.Println("Config modification requested")
	var req map[string]interface{}
	gc.BindJSON(&req)
	tempConfig, _ := ini.Load(app.config_path)
	for section, settings := range req {
		_, err := tempConfig.GetSection(section)
		if section != "restart-program" && err == nil {
			for setting, value := range settings.(map[string]interface{}) {
				tempConfig.Section(section).Key(setting).SetValue(value.(string))
			}
		}
	}
	tempConfig.SaveTo(app.config_path)
	app.debug.Println("Config saved")
	gc.JSON(200, map[string]bool{"success": true})
	if req["restart-program"].(bool) {
		app.info.Println("Restarting...")
		err := app.Restart()
		if err != nil {
			app.err.Printf("Couldn't restart, try restarting manually. (%s)", err)
		}
	}
	app.loadConfig()
	// Reinitialize password validator on config change, as opposed to every applicable request like in python.
	if _, ok := req["password_validation"]; ok {
		app.debug.Println("Reinitializing validator")
		validatorConf := ValidatorConf{
			"characters":           app.config.Section("password_validation").Key("min_length").MustInt(0),
			"uppercase characters": app.config.Section("password_validation").Key("upper").MustInt(0),
			"lowercase characters": app.config.Section("password_validation").Key("lower").MustInt(0),
			"numbers":              app.config.Section("password_validation").Key("number").MustInt(0),
			"special characters":   app.config.Section("password_validation").Key("special").MustInt(0),
		}
		if !app.config.Section("password_validation").Key("enabled").MustBool(false) {
			for key := range validatorConf {
				validatorConf[key] = 0
			}
		}
		app.validator.init(validatorConf)
	}
}

func (app *appContext) Logout(gc *gin.Context) {
	cookie, err := gc.Cookie("refresh")
	if err != nil {
		app.debug.Printf("Couldn't get cookies: %s", err)
		respond(500, "Couldn't fetch cookies", gc)
		return
	}
	app.invalidTokens = append(app.invalidTokens, cookie)
	gc.SetCookie("refresh", "invalid", -1, "/", gc.Request.URL.Hostname(), true, true)
	gc.JSON(200, map[string]bool{"success": true})
}

// func Restart() error {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			os.Exit(0)
// 		}
// 	}()
// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		return err
// 	}
// 	args := os.Args
// 	// for _, key := range args {
// 	// 	fmt.Println(key)
// 	// }
// 	cmd := exec.Command(args[0], args[1:]...)
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	cmd.Dir = cwd
// 	err = cmd.Start()
// 	if err != nil {
// 		return err
// 	}
// 	// cmd.Process.Release()
// 	panic(fmt.Errorf("restarting"))
// }

// func (app *appContext) Restart() error {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			signal.Notify(app.quit, os.Interrupt)
// 			<-app.quit
// 		}
// 	}()
// 	args := os.Args
// 	// After a single restart, args[0] gets messed up and isnt the real executable.
// 	// JFA_DEEP tells the new process its a child, and JFA_EXEC is the real executable
// 	if os.Getenv("JFA_DEEP") == "" {
// 		os.Setenv("JFA_DEEP", "1")
// 		os.Setenv("JFA_EXEC", args[0])
// 	}
// 	env := os.Environ()
// 	err := syscall.Exec(os.Getenv("JFA_EXEC"), []string{""}, env)
// 	if err != nil {
// 		return err
// 	}
// 	panic(fmt.Errorf("restarting"))
// }

// no need to syscall.exec anymore!
func (app *appContext) Restart() error {
	RESTART <- true
	return nil
}
