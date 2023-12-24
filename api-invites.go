package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/itchyny/timefmt-go"
	"github.com/lithammer/shortuuid/v3"
	"github.com/timshannon/badgerhold/v4"
)

const (
	CAPTCHA_VALIDITY = 20 * 60 // Seconds
)

// GenerateInviteCode generates an invite code in the correct format.
func GenerateInviteCode() string {
	// make sure code doesn't begin with number
	inviteCode := shortuuid.New()
	_, err := strconv.Atoi(string(inviteCode[0]))
	for err == nil {
		inviteCode = shortuuid.New()
		_, err = strconv.Atoi(string(inviteCode[0]))
	}
	return inviteCode
}

func (app *appContext) checkInvites() {
	currentTime := time.Now()
	for _, data := range app.storage.GetInvites() {
		captchas := data.Captchas
		captchasExpired := false
		for key, capt := range data.Captchas {
			if time.Now().After(capt.Generated.Add(CAPTCHA_VALIDITY * time.Second)) {
				delete(captchas, key)
				captchasExpired = true
			}
		}
		if captchasExpired {
			data.Captchas = captchas
			app.storage.SetInvitesKey(data.Code, data)
		}

		if data.IsReferral && (!data.UseReferralExpiry || data.ReferrerJellyfinID == "") {
			continue
		}
		expiry := data.ValidTill
		if !currentTime.After(expiry) {
			continue
		}

		app.debug.Printf("Housekeeping: Deleting old invite %s", data.Code)

		// Disable referrals for the user if UseReferralExpiry is enabled, so no new ones are made.
		if data.IsReferral && data.UseReferralExpiry && data.ReferrerJellyfinID != "" {
			user, ok := app.storage.GetEmailsKey(data.ReferrerJellyfinID)
			if ok {
				user.ReferralTemplateKey = ""
				app.storage.SetEmailsKey(data.ReferrerJellyfinID, user)
			}
		}
		notify := data.Notify
		if emailEnabled && app.config.Section("notifications").Key("enabled").MustBool(false) && len(notify) != 0 {
			app.debug.Printf("%s: Expiry notification", data.Code)
			var wait sync.WaitGroup
			for address, settings := range notify {
				if !settings["notify-expiry"] {
					continue
				}
				wait.Add(1)
				go func(addr string) {
					defer wait.Done()
					msg, err := app.email.constructExpiry(data.Code, data, app, false)
					if err != nil {
						app.err.Printf("%s: Failed to construct expiry notification: %v", data.Code, err)
					} else {
						// Check whether notify "address" is an email address of Jellyfin ID
						if strings.Contains(addr, "@") {
							err = app.email.send(msg, addr)
						} else {
							err = app.sendByID(msg, addr)
						}
						if err != nil {
							app.err.Printf("%s: Failed to send expiry notification: %v", data.Code, err)
						} else {
							app.info.Printf("Sent expiry notification to %s", addr)
						}
					}
				}(address)
			}
			wait.Wait()
		}
		app.storage.DeleteInvitesKey(data.Code)

		app.storage.SetActivityKey(shortuuid.New(), Activity{
			Type:       ActivityDeleteInvite,
			SourceType: ActivityDaemon,
			InviteCode: data.Code,
			Value:      data.Label,
			Time:       time.Now(),
		}, nil, false)
	}
}

func (app *appContext) checkInvite(code string, used bool, username string) bool {
	currentTime := time.Now()
	inv, match := app.storage.GetInvitesKey(code)
	if !match {
		return false
	}
	expiry := inv.ValidTill
	if currentTime.After(expiry) {
		app.debug.Printf("Housekeeping: Deleting old invite %s", code)
		notify := inv.Notify
		if emailEnabled && app.config.Section("notifications").Key("enabled").MustBool(false) && len(notify) != 0 {
			app.debug.Printf("%s: Expiry notification", code)
			var wait sync.WaitGroup
			for address, settings := range notify {
				if !settings["notify-expiry"] {
					continue
				}
				wait.Add(1)
				go func(addr string) {
					defer wait.Done()
					msg, err := app.email.constructExpiry(code, inv, app, false)
					if err != nil {
						app.err.Printf("%s: Failed to construct expiry notification: %v", code, err)
					} else {
						// Check whether notify "address" is an email address of Jellyfin ID
						if strings.Contains(addr, "@") {
							err = app.email.send(msg, addr)
						} else {
							err = app.sendByID(msg, addr)
						}
						if err != nil {
							app.err.Printf("%s: Failed to send expiry notification: %v", code, err)
						} else {
							app.info.Printf("Sent expiry notification to %s", addr)
						}
					}
				}(address)
			}
			wait.Wait()
		}
		if inv.IsReferral && inv.ReferrerJellyfinID != "" && inv.UseReferralExpiry {
			user, ok := app.storage.GetEmailsKey(inv.ReferrerJellyfinID)
			if ok {
				user.ReferralTemplateKey = ""
				app.storage.SetEmailsKey(inv.ReferrerJellyfinID, user)
			}
		}
		match = false
		app.storage.DeleteInvitesKey(code)
		app.storage.SetActivityKey(shortuuid.New(), Activity{
			Type:       ActivityDeleteInvite,
			SourceType: ActivityDaemon,
			InviteCode: code,
			Value:      inv.Label,
			Time:       time.Now(),
		}, nil, false)
	} else if used {
		del := false
		newInv := inv
		if newInv.RemainingUses == 1 {
			del = true
			app.storage.DeleteInvitesKey(code)
			app.storage.SetActivityKey(shortuuid.New(), Activity{
				Type:       ActivityDeleteInvite,
				SourceType: ActivityDaemon,
				InviteCode: code,
				Value:      inv.Label,
				Time:       time.Now(),
			}, nil, false)
		} else if newInv.RemainingUses != 0 {
			// 0 means infinite i guess?
			newInv.RemainingUses--
		}
		newInv.UsedBy = append(newInv.UsedBy, []string{username, strconv.FormatInt(currentTime.Unix(), 10)})
		if !del {
			app.storage.SetInvitesKey(code, newInv)
		}
	}
	return match
}

// @Summary Create a new invite.
// @Produce json
// @Param generateInviteDTO body generateInviteDTO true "New invite request object"
// @Success 200 {object} boolResponse
// @Router /invites [post]
// @Security Bearer
// @tags Invites
func (app *appContext) GenerateInvite(gc *gin.Context) {
	var req generateInviteDTO
	app.debug.Println("Generating new invite")
	gc.BindJSON(&req)
	currentTime := time.Now()
	validTill := currentTime.AddDate(0, req.Months, req.Days)
	validTill = validTill.Add(time.Hour*time.Duration(req.Hours) + time.Minute*time.Duration(req.Minutes))
	var invite Invite
	invite.Code = GenerateInviteCode()
	if req.Label != "" {
		invite.Label = req.Label
	}
	if req.UserLabel != "" {
		invite.UserLabel = req.UserLabel
	}
	invite.Created = currentTime
	if req.MultipleUses {
		if req.NoLimit {
			invite.NoLimit = true
		} else {
			invite.RemainingUses = req.RemainingUses
		}
	} else {
		invite.RemainingUses = 1
	}
	invite.UserExpiry = req.UserExpiry
	if invite.UserExpiry {
		invite.UserMonths = req.UserMonths
		invite.UserDays = req.UserDays
		invite.UserHours = req.UserHours
		invite.UserMinutes = req.UserMinutes
	}
	invite.ValidTill = validTill
	if req.SendTo != "" && app.config.Section("invite_emails").Key("enabled").MustBool(false) {
		addressValid := false
		discord := ""
		app.debug.Printf("%s: Sending invite message", invite.Code)
		if discordEnabled && (!strings.Contains(req.SendTo, "@") || strings.HasPrefix(req.SendTo, "@")) {
			users := app.discord.GetUsers(req.SendTo)
			if len(users) == 0 {
				invite.SendTo = fmt.Sprintf("Failed: User not found: \"%s\"", req.SendTo)
			} else if len(users) > 1 {
				invite.SendTo = fmt.Sprintf("Failed: Multiple users found: \"%s\"", req.SendTo)
			} else {
				invite.SendTo = req.SendTo
				addressValid = true
				discord = users[0].User.ID
			}
		} else if emailEnabled {
			addressValid = true
			invite.SendTo = req.SendTo
		}
		if addressValid {
			msg, err := app.email.constructInvite(invite.Code, invite, app, false)
			if err != nil {
				invite.SendTo = fmt.Sprintf("Failed to send to %s", req.SendTo)
				app.err.Printf("%s: Failed to construct invite message: %v", invite.Code, err)
			} else {
				var err error
				if discord != "" {
					err = app.discord.SendDM(msg, discord)
				} else {
					err = app.email.send(msg, req.SendTo)
				}
				if err != nil {
					invite.SendTo = fmt.Sprintf("Failed to send to %s", req.SendTo)
					app.err.Printf("%s: %s: %v", invite.Code, invite.SendTo, err)
				} else {
					app.info.Printf("%s: Sent invite email to \"%s\"", invite.Code, req.SendTo)
				}
			}
		}
	}
	if req.Profile != "" {
		if _, ok := app.storage.GetProfileKey(req.Profile); ok {
			invite.Profile = req.Profile
		} else {
			invite.Profile = "Default"
		}
	}
	app.storage.SetInvitesKey(invite.Code, invite)

	// Record activity
	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityCreateInvite,
		UserID:     "",
		SourceType: ActivityAdmin,
		Source:     gc.GetString("jfId"),
		InviteCode: invite.Code,
		Value:      invite.Label,
		Time:       time.Now(),
	}, gc, false)

	respondBool(200, true, gc)
}

// @Summary Get invites.
// @Produce json
// @Success 200 {object} getInvitesDTO
// @Router /invites [get]
// @Security Bearer
// @tags Invites
func (app *appContext) GetInvites(gc *gin.Context) {
	app.debug.Println("Invites requested")
	currentTime := time.Now()
	app.checkInvites()
	var invites []inviteDTO
	for _, inv := range app.storage.GetInvites() {
		if inv.IsReferral {
			continue
		}
		years, months, days, hours, minutes, _ := timeDiff(inv.ValidTill, currentTime)
		months += years * 12
		invite := inviteDTO{
			Code:        inv.Code,
			Months:      months,
			Days:        days,
			Hours:       hours,
			Minutes:     minutes,
			UserExpiry:  inv.UserExpiry,
			UserMonths:  inv.UserMonths,
			UserDays:    inv.UserDays,
			UserHours:   inv.UserHours,
			UserMinutes: inv.UserMinutes,
			Created:     inv.Created.Unix(),
			Profile:     inv.Profile,
			NoLimit:     inv.NoLimit,
			Label:       inv.Label,
			UserLabel:   inv.UserLabel,
		}
		if len(inv.UsedBy) != 0 {
			invite.UsedBy = map[string]int64{}
			for _, pair := range inv.UsedBy {
				// These used to be stored formatted instead of as a unix timestamp.
				unix, err := strconv.ParseInt(pair[1], 10, 64)
				if err != nil {
					date, err := timefmt.Parse(pair[1], app.datePattern+" "+app.timePattern)
					if err != nil {
						app.err.Printf("Failed to parse usedBy time: %v", err)
					}
					unix = date.Unix()
				}
				invite.UsedBy[pair[0]] = unix
			}
		}
		invite.RemainingUses = 1
		if inv.RemainingUses != 0 {
			invite.RemainingUses = inv.RemainingUses
		}
		if inv.SendTo != "" {
			invite.SendTo = inv.SendTo
		}
		if len(inv.Notify) != 0 {
			// app.err.Printf("%s has notify section: %+v, you are %s\n", inv.Code, inv.Notify, gc.GetString("jfId"))
			var addressOrID string
			if app.config.Section("ui").Key("jellyfin_login").MustBool(false) {
				addressOrID = gc.GetString("jfId")
			} else {
				addressOrID = app.config.Section("ui").Key("email").String()
			}
			if _, ok := inv.Notify[addressOrID]; ok {
				if _, ok = inv.Notify[addressOrID]["notify-expiry"]; ok {
					invite.NotifyExpiry = inv.Notify[addressOrID]["notify-expiry"]
				}
				if _, ok = inv.Notify[addressOrID]["notify-creation"]; ok {
					invite.NotifyCreation = inv.Notify[addressOrID]["notify-creation"]
				}
			}
		}
		invites = append(invites, invite)
	}
	fullProfileList := app.storage.GetProfiles()
	profiles := make([]string, len(fullProfileList))
	if len(profiles) != 0 {
		defaultProfile := app.storage.GetDefaultProfile()
		profiles[0] = defaultProfile.Name
		i := 1
		if len(fullProfileList) > 1 {
			app.storage.db.ForEach(badgerhold.Where("Name").Ne(profiles[0]), func(p *Profile) error {
				profiles[i] = p.Name
				i++
				return nil
			})
		}
	}
	resp := getInvitesDTO{
		Profiles: profiles,
		Invites:  invites,
	}
	gc.JSON(200, resp)
}

// @Summary Set profile for an invite
// @Produce json
// @Param inviteProfileDTO body inviteProfileDTO true "Invite profile object"
// @Success 200 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /invites/profile [post]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) SetProfile(gc *gin.Context) {
	var req inviteProfileDTO
	gc.BindJSON(&req)
	app.debug.Printf("%s: Setting profile to \"%s\"", req.Invite, req.Profile)
	// "" means "Don't apply profile"
	if _, ok := app.storage.GetProfileKey(req.Profile); !ok && req.Profile != "" {
		app.err.Printf("%s: Profile \"%s\" not found", req.Invite, req.Profile)
		respond(500, "Profile not found", gc)
		return
	}
	inv, _ := app.storage.GetInvitesKey(req.Invite)
	inv.Profile = req.Profile
	app.storage.SetInvitesKey(req.Invite, inv)
	respondBool(200, true, gc)
}

// @Summary Set notification preferences for an invite.
// @Produce json
// @Param setNotifyDTO body setNotifyDTO true "Map of invite codes to notification settings objects"
// @Success 200
// @Failure 400 {object} stringResponse
// @Failure 500 {object} stringResponse
// @Router /invites/notify [post]
// @Security Bearer
// @tags Other
func (app *appContext) SetNotify(gc *gin.Context) {
	var req map[string]map[string]bool
	gc.BindJSON(&req)
	changed := false
	for code, settings := range req {
		app.debug.Printf("%s: Notification settings change requested", code)
		invite, ok := app.storage.GetInvitesKey(code)
		if !ok {
			app.err.Printf("%s Notification setting change failed: Invalid code", code)
			respond(400, "Invalid invite code", gc)
			return
		}
		var address string
		jellyfinLogin := app.config.Section("ui").Key("jellyfin_login").MustBool(false)
		if jellyfinLogin {
			var addressAvailable bool = app.getAddressOrName(gc.GetString("jfId")) != ""
			if !addressAvailable {
				app.err.Printf("%s: Couldn't find contact method for admin. Make sure one is set.", code)
				app.debug.Printf("%s: User ID \"%s\"", code, gc.GetString("jfId"))
				respond(500, "Missing user contact method", gc)
				return
			}
			address = gc.GetString("jfId")
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
		if _, ok := settings["notify-expiry"]; ok && invite.Notify[address]["notify-expiry"] != settings["notify-expiry"] {
			invite.Notify[address]["notify-expiry"] = settings["notify-expiry"]
			app.debug.Printf("%s: Set \"notify-expiry\" to %t for %s", code, settings["notify-expiry"], address)
			changed = true
		}
		if _, ok := settings["notify-creation"]; ok && invite.Notify[address]["notify-creation"] != settings["notify-creation"] {
			invite.Notify[address]["notify-creation"] = settings["notify-creation"]
			app.debug.Printf("%s: Set \"notify-creation\" to %t for %s", code, settings["notify-creation"], address)
			changed = true
		}
		if changed {
			app.storage.SetInvitesKey(code, invite)
		}
	}
}

// @Summary Delete an invite.
// @Produce json
// @Param deleteInviteDTO body deleteInviteDTO true "Delete invite object"
// @Success 200 {object} boolResponse
// @Failure 400 {object} stringResponse
// @Router /invites [delete]
// @Security Bearer
// @tags Invites
func (app *appContext) DeleteInvite(gc *gin.Context) {
	var req deleteInviteDTO
	gc.BindJSON(&req)
	app.debug.Printf("%s: Deletion requested", req.Code)
	inv, ok := app.storage.GetInvitesKey(req.Code)
	if ok {
		app.storage.DeleteInvitesKey(req.Code)

		// Record activity
		app.storage.SetActivityKey(shortuuid.New(), Activity{
			Type:       ActivityDeleteInvite,
			SourceType: ActivityAdmin,
			Source:     gc.GetString("jfId"),
			InviteCode: req.Code,
			Value:      inv.Label,
			Time:       time.Now(),
		}, gc, false)

		app.info.Printf("%s: Invite deleted", req.Code)
		respondBool(200, true, gc)
		return
	}
	app.err.Printf("%s: Deletion failed: Invalid code", req.Code)
	respond(400, "Code doesn't exist", gc)
}
