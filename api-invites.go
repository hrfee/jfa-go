package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
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

// checkInvites performs general housekeeping on invites, i.e. deleting expired ones and cleaning captcha data.
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
		app.deleteExpiredInvite(data)
	}
}

// checkInvite checks the validity of a specific invite, optionally removing it if invalid(ated).
func (app *appContext) checkInvite(code string, used bool, username string) bool {
	currentTime := time.Now()
	inv, match := app.storage.GetInvitesKey(code)
	if !match {
		return false
	}
	expiry := inv.ValidTill
	if currentTime.After(expiry) {
		app.deleteExpiredInvite(inv)
		match = false
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

func (app *appContext) deleteExpiredInvite(data Invite) {
	app.debug.Printf(lm.DeleteOldInvite, data.Code)

	// Disable referrals for the user if UseReferralExpiry is enabled, so no new ones are made.
	if data.IsReferral && data.UseReferralExpiry && data.ReferrerJellyfinID != "" {
		user, ok := app.storage.GetEmailsKey(data.ReferrerJellyfinID)
		if ok {
			user.ReferralTemplateKey = ""
			app.storage.SetEmailsKey(data.ReferrerJellyfinID, user)
		}
	}
	wait := app.sendAdminExpiryNotification(data)
	app.storage.DeleteInvitesKey(data.Code)

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityDeleteInvite,
		SourceType: ActivityDaemon,
		InviteCode: data.Code,
		Value:      data.Label,
		Time:       time.Now(),
	}, nil, false)

	if wait != nil {
		wait.Wait()
	}
}

func (app *appContext) sendAdminExpiryNotification(data Invite) *sync.WaitGroup {
	notify := data.Notify
	if !emailEnabled || !app.config.Section("notifications").Key("enabled").MustBool(false) || len(notify) != 0 {
		return nil
	}
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
				app.err.Printf(lm.FailedConstructExpiryAdmin, data.Code, err)
			} else {
				// Check whether notify "address" is an email address or Jellyfin ID
				if strings.Contains(addr, "@") {
					err = app.email.send(msg, addr)
				} else {
					err = app.sendByID(msg, addr)
				}
				if err != nil {
					app.err.Printf(lm.FailedSendExpiryAdmin, data.Code, addr, err)
				} else {
					app.info.Printf(lm.SentExpiryAdmin, data.Code, addr)
				}
			}
		}(address)
	}
	return &wait
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
	app.debug.Println(lm.GenerateInvite)
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
	if req.SendTo != "" {
		if !(app.config.Section("invite_emails").Key("enabled").MustBool(false)) {
			app.err.Printf(lm.FailedSendInviteMessage, invite.Code, req.SendTo, errors.New(lm.InviteMessagesDisabled))
		} else {
			addressValid := false
			discord := ""
			if discordEnabled && (!strings.Contains(req.SendTo, "@") || strings.HasPrefix(req.SendTo, "@")) {
				users := app.discord.GetUsers(req.SendTo)
				if len(users) == 0 {
					invite.SendTo = fmt.Sprintf(lm.FailedSendToTooltipNoUser, req.SendTo)
				} else if len(users) > 1 {
					invite.SendTo = fmt.Sprintf(lm.FailedSendToTooltipMultiUser, req.SendTo)
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
					// Slight misuse of the template
					invite.SendTo = fmt.Sprintf(lm.FailedConstructInviteMessage, req.SendTo, err)

					app.err.Printf(lm.FailedConstructInviteMessage, invite.Code, err)
				} else {
					var err error
					if discord != "" {
						err = app.discord.SendDM(msg, discord)
					} else {
						err = app.email.send(msg, req.SendTo)
					}
					if err != nil {
						invite.SendTo = fmt.Sprintf(lm.FailedSendInviteMessage, invite.Code, req.SendTo, err)
						app.err.Println(invite.SendTo)
					} else {
						app.info.Printf(lm.SentInviteMessage, invite.Code, req.SendTo)
					}
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

// @Summary Get the number of invites stored in the database.
// @Produce json
// @Success 200 {object} PageCountDTO
// @Router /invites/count [get]
// @Security Bearer
// @tags Invites
func (app *appContext) GetInviteCount(gc *gin.Context) {
	resp := PageCountDTO{}
	var err error
	resp.Count, err = app.storage.db.Count(&Invite{}, badgerhold.Where("IsReferral").Eq(false))
	if err != nil {
		resp.Count = 0
	}
	gc.JSON(200, resp)
}

// @Summary Get the number of invites stored in the database that have been used (but are still valid).
// @Produce json
// @Success 200 {object} PageCountDTO
// @Router /invites/count [get]
// @Security Bearer
// @tags Invites
func (app *appContext) GetInviteUsedCount(gc *gin.Context) {
	resp := PageCountDTO{}
	var err error
	resp.Count, err = app.storage.db.Count(&Invite{}, badgerhold.Where("IsReferral").Eq(false).And("UsedBy").MatchFunc(func(ra *badgerhold.RecordAccess) (bool, error) {
		field := ra.Field()
		switch usedBy := field.(type) {
		case [][]string:
			return len(usedBy) > 0, nil
		default:
			return false, nil
		}
	}))
	if err != nil {
		resp.Count = 0
	}
	gc.JSON(200, resp)
}

// @Summary Get invites.
// @Produce json
// @Success 200 {object} getInvitesDTO
// @Router /invites [get]
// @Security Bearer
// @tags Invites
func (app *appContext) GetInvites(gc *gin.Context) {
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
						app.err.Printf(lm.FailedParseTime, err)
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
	resp := getInvitesDTO{
		Invites: invites,
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
// @tags Invites
func (app *appContext) SetProfile(gc *gin.Context) {
	var req inviteProfileDTO
	gc.BindJSON(&req)
	// "" means "Don't apply profile"
	if _, ok := app.storage.GetProfileKey(req.Profile); !ok && req.Profile != "" {
		app.err.Printf(lm.FailedGetProfile, req.Profile)
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
		invite, ok := app.storage.GetInvitesKey(code)
		if !ok {
			msg := fmt.Sprintf(lm.InvalidInviteCode, code)
			app.err.Println(msg)
			respond(400, msg, gc)
			return
		}
		var address string
		jellyfinLogin := app.config.Section("ui").Key("jellyfin_login").MustBool(false)
		if jellyfinLogin {
			var addressAvailable bool = app.getAddressOrName(gc.GetString("jfId")) != ""
			if !addressAvailable {
				app.err.Printf(lm.FailedGetContactMethod, gc.GetString("jfId"))
				respond(500, fmt.Sprintf(lm.FailedGetContactMethod, "admin"), gc)
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
		for _, notifyType := range []string{"notify-expiry", "notify-creation"} {
			if _, ok := settings[notifyType]; ok && invite.Notify[address][notifyType] != settings[notifyType] {
				invite.Notify[address][notifyType] = settings[notifyType]
				app.debug.Printf(lm.SetAdminNotify, notifyType, settings[notifyType], address)
				changed = true
			}
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

		app.info.Printf(lm.DeleteInvite, req.Code)
		respondBool(200, true, gc)
		return
	}
	app.err.Printf(lm.FailedDeleteInvite, req.Code, "invalid code")
	respond(400, "Code doesn't exist", gc)
}
