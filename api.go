package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/knz/strtime"
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
		fmt.Println("Expiry:", expiry)
		if current_time.After(expiry) {
			// NOTIFICATIONS
			notify := data.Notify
			if ctx.config.Section("notifications").Key("enabled").MustBool(false) && len(notify) != 0 {
				for address, settings := range notify {
					if settings["notify-expiry"] {
						if ctx.email.constructExpiry(invCode, data, ctx) != nil {
							fmt.Println("failed expiry construct")
						} else {
							if ctx.email.send(address, ctx) != nil {
								fmt.Println("failed expiry send")
							}
						}
					}
				}
			}
			changed = true
			fmt.Println("Deleting:", invCode)
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

// POST
type newUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Code     string `json:"code"`
}

func (ctx *appContext) NewUser(gc *gin.Context) {
	var req newUserReq
	gc.BindJSON(&req)
	if !ctx.checkInvite(req.Code, false, "") {
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
		fmt.Println("invalid")
		gc.JSON(200, validation)
		gc.Abort()
		return
	}
	existingUser, _, _ := ctx.jf.userByName(req.Username, false)
	if existingUser != nil {
		respond(401, fmt.Sprintf("User already exists named %s", req.Username), gc)
		return
	}
	user, status, err := ctx.jf.newUser(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		respond(401, "Unknown error", gc)
		return
	}
	ctx.checkInvite(req.Code, true, req.Username)
	invite := ctx.storage.invites[req.Code]
	if ctx.config.Section("notifications").Key("enabled").MustBool(false) {
		for address, settings := range invite.Notify {
			if settings["notify-creation"] {
				if ctx.email.constructCreated(req.Code, req.Username, req.Email, invite, ctx) != nil {
					fmt.Println("created template failed")
				} else if ctx.email.send(address, ctx) != nil {
					fmt.Println("created send failed")
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
			fmt.Printf("Failed to set user policy")
		}
	}
	if len(ctx.storage.configuration) != 0 && len(ctx.storage.displayprefs) != 0 {
		status, err = ctx.jf.setConfiguration(id, ctx.storage.configuration)
		if (status == 200 || status == 204) && err != nil {
			status, err = ctx.jf.setDisplayPreferences(id, ctx.storage.displayprefs)
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
	RemainingUses int    `json:remaining-uses"`
}

func (ctx *appContext) GenerateInvite(gc *gin.Context) {
	var req generateInviteReq
	ctx.storage.loadInvites()
	gc.BindJSON(&req)
	current_time := time.Now()
	fmt.Println(req.Days, req.Hours, req.Minutes)
	valid_till := current_time.AddDate(0, 0, req.Days)
	valid_till = valid_till.Add(time.Hour*time.Duration(req.Hours) + time.Minute*time.Duration(req.Minutes))
	invite_code, _ := uuid.NewRandom()
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
		invite.Email = req.Email
		if err := ctx.email.constructInvite(invite_code.String(), invite, ctx); err != nil {
			fmt.Println("error sending:", err)
			invite.Email = fmt.Sprintf("Failed to send to %s", req.Email)
		} else if err := ctx.email.send(req.Email, ctx); err != nil {
			fmt.Println("error sending:", err)
			invite.Email = fmt.Sprintf("Failed to send to %s", req.Email)
		}
	}
	ctx.storage.invites[invite_code.String()] = invite
	fmt.Println("INVITES FROM API:", ctx.storage.invites)
	ctx.storage.storeInvites()
	fmt.Println("New inv")
	gc.JSON(200, map[string]bool{"success": true})
}

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
