package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/knz/strtime"
	"github.com/lithammer/shortuuid/v3"
	"gopkg.in/ini.v1"
	"time"
)

func (ctx *appContext) loadStrftime() {
	ctx.datePattern = ctx.config.Section("email").Key("date_format").String()
	ctx.timePattern = `%H:%M`
	if val, _ := ctx.config.Section("email").Key("use_24h").Bool(); !val {
		ctx.timePattern = `%I:%M %p`
	}
	return
}

func (ctx *appContext) prettyTime(dt time.Time) (date, time string) {
	date, _ = strtime.Strftime(dt, ctx.datePattern)
	time, _ = strtime.Strftime(dt, ctx.timePattern)
	return
}

func (ctx *appContext) formatDatetime(dt time.Time) string {
	d, t := ctx.prettyTime(dt)
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

func (ctx *appContext) checkInvite(code string, used bool, username string) bool {
	current_time := time.Now()
	ctx.storage.loadInvites()
	match := false
	changed := false
	for invCode, data := range ctx.storage.invites {
		expiry := data.ValidTill
		if current_time.After(expiry) {
			ctx.debug.Printf("Housekeeping: Deleting old invite %s", code)
			notify := data.Notify
			if ctx.config.Section("notifications").Key("enabled").MustBool(false) && len(notify) != 0 {
				ctx.debug.Printf("%s: Expiry notification", code)
				for address, settings := range notify {
					if settings["notify-expiry"] {
						if ctx.email.constructExpiry(invCode, data, ctx) != nil {
							ctx.err.Printf("%s: Failed to construct expiry notification", code)
						} else if ctx.email.send(address, ctx) != nil {
							ctx.err.Printf("%s: Failed to send expiry notification", code)
						} else {
							ctx.info.Printf("Sent expiry notification to %s", address)
						}
					}
				}
			}
			changed = true
			delete(ctx.storage.invites, invCode)
		} else if invCode == code {
			match = true
			if used {
				changed = true
				del := false
				newInv := data
				if newInv.RemainingUses == 1 {
					del = true
					delete(ctx.storage.invites, invCode)
				} else if newInv.RemainingUses != 0 {
					// 0 means infinite i guess?
					newInv.RemainingUses -= 1
				}
				newInv.UsedBy = append(newInv.UsedBy, []string{username, ctx.formatDatetime(current_time)})
				if !del {
					ctx.storage.invites[invCode] = newInv
				}
			}
		}
	}
	if changed {
		ctx.storage.storeInvites()
	}
	return match
}

// Routes from now on!

type newUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Code     string `json:"code"`
}

func (ctx *appContext) NewUser(gc *gin.Context) {
	var req newUserReq
	gc.BindJSON(&req)
	ctx.debug.Printf("%s: New user attempt", req.Code)
	if !ctx.checkInvite(req.Code, false, "") {
		ctx.info.Printf("%s New user failed: invalid code", req.Code)
		gc.JSON(401, map[string]bool{"success": false})
		gc.Abort()
		return
	}
	validation := ctx.validator.validate(req.Password)
	valid := true
	for _, val := range validation {
		if !val {
			valid = false
		}
	}
	if !valid {
		// 200 bcs idk what i did in js
		ctx.info.Printf("%s New user failed: Invalid password", req.Code)
		gc.JSON(200, validation)
		gc.Abort()
		return
	}
	existingUser, _, _ := ctx.jf.userByName(req.Username, false)
	if existingUser != nil {
		msg := fmt.Sprintf("User already exists named %s", req.Username)
		ctx.info.Printf("%s New user failed: %s", req.Code, msg)
		respond(401, msg, gc)
		return
	}
	user, status, err := ctx.jf.newUser(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		ctx.err.Printf("%s New user failed: Jellyfin responded with %d", req.Code, status)
		respond(401, "Unknown error", gc)
		return
	}
	ctx.checkInvite(req.Code, true, req.Username)
	invite := ctx.storage.invites[req.Code]
	if ctx.config.Section("notifications").Key("enabled").MustBool(false) {
		for address, settings := range invite.Notify {
			if settings["notify-creation"] {
				if ctx.email.constructCreated(req.Code, req.Username, req.Email, invite, ctx) != nil {
					ctx.err.Printf("%s: Failed to construct user creation notification", req.Code)
					ctx.debug.Printf("%s: Error: %s", req.Code, err)
				} else if ctx.email.send(address, ctx) != nil {
					ctx.err.Printf("%s: Failed to send user creation notification", req.Code)
					ctx.debug.Printf("%s: Error: %s", req.Code, err)
				} else {
					ctx.info.Printf("%s: Sent user creation notification to %s", req.Code, address)
				}
			}
		}
	}
	var id string
	if user["Id"] != nil {
		id = user["Id"].(string)
	}
	if len(ctx.storage.policy) != 0 {
		status, err = ctx.jf.setPolicy(id, ctx.storage.policy)
		if !(status == 200 || status == 204) {
			ctx.err.Printf("%s: Failed to set user policy: Code %d", req.Code, status)
		}
	}
	if len(ctx.storage.configuration) != 0 && len(ctx.storage.displayprefs) != 0 {
		status, err = ctx.jf.setConfiguration(id, ctx.storage.configuration)
		if (status == 200 || status == 204) && err == nil {
			status, err = ctx.jf.setDisplayPreferences(id, ctx.storage.displayprefs)
		} else {
			ctx.err.Printf("%s: Failed to set configuration template: Code %d", req.Code, status)
		}
	}
	if ctx.config.Section("password_resets").Key("enabled").MustBool(false) {
		ctx.storage.emails[id] = req.Email
		ctx.storage.storeEmails()
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

func (ctx *appContext) GenerateInvite(gc *gin.Context) {
	var req generateInviteReq
	ctx.debug.Println("Generating new invite")
	ctx.storage.loadInvites()
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
	if req.Email != "" && ctx.config.Section("invite_emails").Key("enabled").MustBool(false) {
		ctx.debug.Printf("%s: Sending invite email", invite_code)
		invite.Email = req.Email
		if err := ctx.email.constructInvite(invite_code, invite, ctx); err != nil {
			invite.Email = fmt.Sprintf("Failed to send to %s", req.Email)
			ctx.err.Printf("%s: Failed to construct invite email", invite_code)
			ctx.debug.Printf("%s: Error: %s", invite_code, err)
		} else if err := ctx.email.send(req.Email, ctx); err != nil {
			invite.Email = fmt.Sprintf("Failed to send to %s", req.Email)
			ctx.err.Printf("%s: %s", invite_code, invite.Email)
			ctx.debug.Printf("%s: Error: %s", invite_code, err)
		} else {
			ctx.info.Printf("%s: Sent invite email to %s", invite_code, req.Email)
		}
	}
	ctx.storage.invites[invite_code] = invite
	ctx.storage.storeInvites()
	gc.JSON(200, map[string]bool{"success": true})
}

// logged up to here!

func (ctx *appContext) GetInvites(gc *gin.Context) {
	current_time := time.Now()
	// checking one checks all of them
	ctx.storage.loadInvites()
	for key := range ctx.storage.invites {
		ctx.checkInvite(key, false, "")
		break
	}
	var invites []map[string]interface{}
	for code, inv := range ctx.storage.invites {
		_, _, days, hours, minutes, _ := timeDiff(inv.ValidTill, current_time)
		invite := make(map[string]interface{})
		invite["code"] = code
		invite["days"] = days
		invite["hours"] = hours
		invite["minutes"] = minutes
		invite["created"] = ctx.formatDatetime(inv.Created)
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
			if ctx.config.Section("ui").Key("jellyfin_login").MustBool(false) {
				ctx.storage.loadEmails()
				address = ctx.storage.emails[gc.GetString("jfId")].(string)
			} else {
				address = ctx.config.Section("ui").Key("email").String()
			}
			if _, ok := inv.Notify[address]; ok {
				for _, notify_type := range []string{"notify-expiry", "notify-creation"} {
					if _, ok = inv.Notify[notify_type]; ok {
						invite[notify_type] = inv.Notify[address][notify_type]
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

func (ctx *appContext) SetNotify(gc *gin.Context) {
	var req map[string]notifySetting
	gc.BindJSON(&req)
	changed := false
	for code, settings := range req {
		ctx.storage.loadInvites()
		ctx.storage.loadEmails()
		invite, ok := ctx.storage.invites[code]
		if !ok {
			gc.JSON(400, map[string]string{"error": "Invalid invite code"})
			gc.Abort()
			return
		}
		var address string
		if ctx.config.Section("ui").Key("jellyfin_login").MustBool(false) {
			var ok bool
			address, ok = ctx.storage.emails[gc.GetString("jfId")].(string)
			if !ok {
				gc.JSON(500, map[string]string{"error": "Missing user email"})
				gc.Abort()
				return
			}
		} else {
			address = ctx.config.Section("ui").Key("email").String()
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
			changed = true
		}
		if invite.Notify[address]["notify-creation"] != settings.NotifyCreation {
			invite.Notify[address]["notify-creation"] = settings.NotifyCreation
			changed = true
		}
		if changed {
			ctx.storage.invites[code] = invite
		}
	}
	if changed {
		ctx.storage.storeInvites()
	}
}

type deleteReq struct {
	Code string `json:"code"`
}

func (ctx *appContext) DeleteInvite(gc *gin.Context) {
	var req deleteReq
	gc.BindJSON(&req)
	var ok bool
	fmt.Println(req.Code)
	fmt.Println(ctx.storage.invites[req.Code])
	_, ok = ctx.storage.invites[req.Code]
	if ok {
		fmt.Println("deleting invite")
		delete(ctx.storage.invites, req.Code)
		ctx.storage.storeInvites()
		gc.JSON(200, map[string]bool{"success": true})
		return
	}
	respond(401, "Code doesn't exist", gc)
}

type userResp struct {
	UserList []respUser `json:"users"`
}

type respUser struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

func (ctx *appContext) GetUsers(gc *gin.Context) {
	var resp userResp
	resp.UserList = []respUser{}
	users, status, err := ctx.jf.getUsers(false)
	if !(status == 200 || status == 204) || err != nil {
		respond(500, "Couldn't get users", gc)
		return
	}
	for _, jfUser := range users {
		var user respUser
		user.Name = jfUser["Name"].(string)
		if email, ok := ctx.storage.emails[jfUser["Id"].(string)]; ok {
			user.Email = email.(string)
		}
		resp.UserList = append(resp.UserList, user)
	}
	gc.JSON(200, resp)
}

func (ctx *appContext) ModifyEmails(gc *gin.Context) {
	var req map[string]string
	gc.BindJSON(&req)
	users, status, err := ctx.jf.getUsers(false)
	if !(status == 200 || status == 204) || err != nil {
		respond(500, "Couldn't get users", gc)
		return
	}
	for _, jfUser := range users {
		if address, ok := req[jfUser["Name"].(string)]; ok {
			ctx.storage.emails[jfUser["Id"].(string)] = address
		}
	}
	ctx.storage.storeEmails()
	gc.JSON(200, map[string]bool{"success": true})
}

type defaultsReq struct {
	Username   string `json:"username"`
	Homescreen bool   `json:"homescreen"`
}

func (ctx *appContext) SetDefaults(gc *gin.Context) {
	var req defaultsReq
	gc.BindJSON(&req)
	user, status, err := ctx.jf.userByName(req.Username, false)
	if !(status == 200 || status == 204) || err != nil {
		respond(500, "Couldn't get user", gc)
		return
	}
	userId := user["Id"].(string)
	policy := user["Policy"].(map[string]interface{})
	ctx.storage.policy = policy
	ctx.storage.storePolicy()
	if req.Homescreen {
		configuration := user["Configuration"].(map[string]interface{})
		var displayprefs map[string]interface{}
		displayprefs, status, err = ctx.jf.getDisplayPreferences(userId)
		if !(status == 200 || status == 204) || err != nil {
			respond(500, "Couldn't get displayprefs", gc)
			return
		}
		ctx.storage.configuration = configuration
		ctx.storage.displayprefs = displayprefs
		ctx.storage.storeConfiguration()
		ctx.storage.storeDisplayprefs()
	}
	gc.JSON(200, map[string]bool{"success": true})
}

func (ctx *appContext) GetConfig(gc *gin.Context) {
	resp := map[string]interface{}{}
	for section, settings := range ctx.configBase {
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
						fmt.Println(resp[section].(map[string]interface{})[key].(map[string]interface{}))
						dataType := resp[section].(map[string]interface{})[key].(map[string]interface{})["type"].(string)
						configKey := ctx.config.Section(section).Key(key)
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

func (ctx *appContext) ModifyConfig(gc *gin.Context) {
	var req map[string]interface{}
	gc.BindJSON(&req)
	tempConfig, _ := ini.Load(ctx.config_path)
	for section, settings := range req {
		_, err := tempConfig.GetSection(section)
		if section != "restart-program" && err == nil {
			for setting, value := range settings.(map[string]interface{}) {
				tempConfig.Section(section).Key(setting).SetValue(value.(string))
			}
		}
	}
	tempConfig.SaveTo(ctx.config_path)
	gc.JSON(200, map[string]bool{"success": true})
	ctx.loadConfig()
}
