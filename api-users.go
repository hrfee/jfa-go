package main

import (
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/hrfee/jfa-go/jellyseerr"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/hrfee/mediabrowser"
	"github.com/lithammer/shortuuid/v3"
	"github.com/timshannon/badgerhold/v4"
)

// @Summary Creates a new Jellyfin user without an invite.
// @Produce json
// @Param newUserDTO body newUserDTO true "New user request object"
// @Success 200
// @Router /user [post]
// @Security Bearer
// @tags Users
func (app *appContext) NewUserFromAdmin(gc *gin.Context) {
	respondUser := func(code int, user, email bool, msg string, gc *gin.Context) {
		resp := newUserResponse{
			User:  user,
			Email: email,
			Error: msg,
		}
		gc.JSON(code, resp)
		gc.Abort()
	}
	var req newUserDTO
	gc.BindJSON(&req)

	profile := app.storage.GetDefaultProfile()
	if req.Profile != "" && req.Profile != "none" {
		if p, ok := app.storage.GetProfileKey(req.Profile); ok {
			profile = p
		}
	}
	nu /*wg*/, _ := app.NewUserPostVerification(NewUserParams{
		Req:                 req,
		SourceType:          ActivityAdmin,
		Source:              gc.GetString("jfId"),
		ContextForIPLogging: gc,
		Profile:             &profile,
	})
	if !nu.Success {
		nu.Log()
	}

	if emailEnabled && req.Email != "" {
		emailStore := EmailAddress{
			Addr:    req.Email,
			Contact: true,
		}
		app.storage.SetEmailsKey(nu.User.ID, emailStore)
	}

	welcomeMessageSentIfNecessary := true
	if nu.Created {
		welcomeMessageSentIfNecessary = !app.WelcomeNewUser(nu.User, time.Time{})
	}

	respondUser(nu.Status, nu.Created, welcomeMessageSentIfNecessary, nu.Message, gc)
	// These don't need to complete anytime soon
	// wg.Wait()
}

// @Summary Creates a new Jellyfin user via invite code
// @Produce json
// @Param newUserDTO body newUserDTO true "New user request object"
// @Success 200 {object} PasswordValidation
// @Failure 400 {object} PasswordValidation
// @Router /user/invite [post]
// @tags Users
func (app *appContext) NewUserFromInvite(gc *gin.Context) {
	/*
		-- STEPS --
		- Validate CAPTCHA
		- Validate Invite
		- Validate Password
		- a) Discord  (Require, Verify, ExistingUser, ApplyRole)
		  b) Telegram (Require, Verify, ExistingUser)
		  c) Matrix   (Require, Verify, ExistingUser)
		  d) Email    (Require, Verify, ExistingUser) (only occurs here, PWRs are sent by mail so do this on their own, kinda)
	*/
	var req newUserDTO
	gc.BindJSON(&req)

	// Validate CAPTCHA
	if app.config.Section("captcha").Key("enabled").MustBool(false) && !app.verifyCaptcha(req.Code, req.CaptchaID, req.CaptchaText, false) {
		app.info.Printf(lm.FailedCreateUser, lm.Jellyfin, req.Username, lm.IncorrectCaptcha)
		respond(400, "errorCaptcha", gc)
		return
	}
	// Validate Invite
	if !app.checkInvite(req.Code, false, "") {
		app.info.Printf(lm.FailedCreateUser, lm.Jellyfin, req.Username, fmt.Sprintf(lm.InvalidInviteCode, req.Code))
		respond(401, "errorInvalidCode", gc)
		return
	}
	// Validate Password
	validation := app.validator.validate(req.Password)
	valid := true
	for _, val := range validation {
		if !val {
			valid = false
			break
		}
	}
	if !valid {
		// 200 bcs idk what i did in js
		gc.JSON(200, validation)
		return
	}
	completeContactMethods := make([]ContactMethodKey, len(app.contactMethods))
	for i, cm := range app.contactMethods {
		completeContactMethods[i].PIN = cm.PIN(req)
		if completeContactMethods[i].PIN == "" {
			if cm.Required() {
				app.info.Printf(lm.FailedLinkUser, cm.Name(), "?", req.Code, lm.AccountUnverified)
				respond(401, fmt.Sprintf("error%sVerification", cm.Name()), gc)
				return
			}
		} else {
			completeContactMethods[i].User, completeContactMethods[i].Verified = cm.UserVerified(completeContactMethods[i].PIN)
			if !completeContactMethods[i].Verified {
				app.info.Printf(lm.FailedLinkUser, cm.Name(), "?", req.Code, fmt.Sprintf(lm.InvalidPIN, completeContactMethods[i].PIN))
				respond(401, "errorInvalidPIN", gc)
				return
			}
			if cm.UniqueRequired() && cm.Exists(completeContactMethods[i].User) {
				app.debug.Printf(lm.FailedLinkUser, cm.Name(), completeContactMethods[i].User.Name(), req.Code, lm.AccountLinked)
				respond(400, "errorAccountLinked", gc)
				return
			}
			if err := cm.PostVerificationTasks(completeContactMethods[i].PIN, completeContactMethods[i].User); err != nil {
				app.err.Printf(lm.FailedLinkUser, cm.Name(), completeContactMethods[i].User.Name(), req.Code, err)
			}
		}
	}
	// Email (FIXME: Potentially make this a ContactMethodLinker)
	if emailEnabled {
		// Require
		if app.config.Section("email").Key("required").MustBool(false) && !strings.Contains(req.Email, "@") {
			respond(400, "errorNoEmail", gc)
			return
		}
		// ExistingUser
		if app.config.Section("email").Key("require_unique").MustBool(false) && req.Email != "" && app.EmailAddressExists(req.Email) {
			respond(400, "errorEmailLinked", gc)
			return
		}
		// Verify
		if app.config.Section("email_confirmation").Key("enabled").MustBool(false) {
			claims := jwt.MapClaims{
				"valid":  true,
				"invite": req.Code,
				"exp":    time.Now().Add(30 * time.Minute).Unix(),
				"type":   "confirmation",
			}
			tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			key, err := tk.SignedString([]byte(os.Getenv("JFA_SECRET")))
			if err != nil {
				app.err.Printf(lm.FailedSignJWT, err)
				respond(500, "errorUnknown", gc)
				return
			}
			if app.ConfirmationKeys == nil {
				app.ConfirmationKeys = map[string]map[string]ConfirmationKey{}
			}
			cKeys, ok := app.ConfirmationKeys[req.Code]
			if !ok {
				cKeys = map[string]ConfirmationKey{}
			}
			cKeys[key] = ConfirmationKey{
				newUserDTO:             req,
				completeContactMethods: completeContactMethods,
			}
			app.confirmationKeysLock.Lock()
			app.ConfirmationKeys[req.Code] = cKeys
			app.confirmationKeysLock.Unlock()

			app.debug.Printf(lm.EmailConfirmationRequired, req.Username)
			respond(401, "confirmEmail", gc)
			msg, err := app.email.constructConfirmation(req.Code, req.Username, key, app, false)
			if err != nil {
				app.err.Printf(lm.FailedConstructConfirmationEmail, req.Code, err)
			} else if err := app.email.send(msg, req.Email); err != nil {
				app.err.Printf(lm.FailedSendConfirmationEmail, req.Code, req.Email, err)
			} else {
				app.debug.Printf(lm.SentConfirmationEmail, req.Code, req.Email)
			}
			return
		}
	}

	invite, _ := app.storage.GetInvitesKey(req.Code)

	sourceType, source := invite.Source()

	var profile *Profile = nil
	if invite.Profile != "" {
		p, ok := app.storage.GetProfileKey(invite.Profile)
		if !ok {
			app.debug.Printf(lm.FailedGetProfile+lm.FallbackToDefault, invite.Profile)
			p = app.storage.GetDefaultProfile()
		}
		profile = &p
	}

	nu /*wg*/, _ := app.NewUserPostVerification(NewUserParams{
		Req:                 req,
		SourceType:          sourceType,
		Source:              source,
		ContextForIPLogging: gc,
		Profile:             profile,
	})
	if !nu.Success {
		nu.Log()
	}
	if !nu.Created {
		respond(nu.Status, nu.Message, gc)
		return
	}
	app.checkInvite(req.Code, true, req.Username)

	app.PostNewUserFromInvite(nu, ConfirmationKey{newUserDTO: req, completeContactMethods: completeContactMethods}, profile, invite)

	/*responseFunc, logFunc, success := app.NewUser(req, false, gc)
	if !success {
		logFunc()
		responseFunc(gc)
		return
	}*/
	code := 200
	for _, val := range validation {
		if !val {
			code = 400
		}
	}

	gc.JSON(code, validation)
	// These don't need to complete anytime soon
	// wg.Wait()
}

// PostNewUserFromInvite attaches user details (e.g. contact method details) to a new user once they've been created from an invite.
func (app *appContext) PostNewUserFromInvite(nu NewUserData, req ConfirmationKey, profile *Profile, invite Invite) {
	nonEmailContactMethodEnabled := false
	for i, c := range req.completeContactMethods {
		if c.Verified {
			c.User.SetAllowContactFromDTO(req.newUserDTO)
			if c.User.AllowContact() {
				nonEmailContactMethodEnabled = true
			}
			app.contactMethods[i].DeleteVerifiedToken(c.PIN)
			c.User.SetJellyfin(nu.User.ID)
			c.User.Store(&(app.storage))
		}
	}

	referralsEnabled := profile != nil && profile.ReferralTemplateKey != "" && app.config.Section("user_page").Key("enabled").MustBool(false) && app.config.Section("user_page").Key("referrals").MustBool(false)

	if (emailEnabled && req.Email != "") || invite.UserLabel != "" || referralsEnabled {
		emailStore := EmailAddress{
			Addr:    req.Email,
			Contact: (req.Email != ""),
			Label:   invite.UserLabel,
		}
		if profile != nil {
			profile.ReferralTemplateKey = profile.ReferralTemplateKey
		}
		/// Ensures at least one contact method is enabled.
		if nonEmailContactMethodEnabled {
			emailStore.Contact = req.EmailContact
		}
		app.storage.SetEmailsKey(nu.User.ID, emailStore)
	}
	if emailEnabled {
		if app.config.Section("notifications").Key("enabled").MustBool(false) {
			for address, settings := range invite.Notify {
				if !settings["notify-creation"] {
					continue
				}
				go func(addr string) {
					msg, err := app.email.constructCreated(req.Code, req.Username, req.Email, invite, app, false)
					if err != nil {
						app.err.Printf(lm.FailedConstructCreationAdmin, req.Code, err)
					} else {
						// Check whether notify "addr" is an email address of Jellyfin ID
						if strings.Contains(addr, "@") {
							err = app.email.send(msg, addr)
						} else {
							err = app.sendByID(msg, addr)
						}
						if err != nil {
							app.err.Printf(lm.FailedSendCreationAdmin, req.Code, addr, err)
						} else {
							app.debug.Printf(lm.SentCreationAdmin, req.Code, addr)
						}
					}
				}(address)
			}
		}
	}

	if referralsEnabled {
		refInv := Invite{}
		/*err := */ app.storage.db.Get(profile.ReferralTemplateKey, &refInv)
		// If UseReferralExpiry is enabled, create the ref now so the clock starts ticking
		if refInv.UseReferralExpiry {
			refInv.Code = GenerateInviteCode()
			expiryDelta := refInv.ValidTill.Sub(refInv.Created)
			refInv.Created = time.Now()
			refInv.ValidTill = refInv.Created.Add(expiryDelta)
			refInv.IsReferral = true
			refInv.ReferrerJellyfinID = nu.User.ID
			app.storage.SetInvitesKey(refInv.Code, refInv)
		}
	}

	expiry := time.Time{}
	if invite.UserExpiry {
		expiry = time.Now().AddDate(0, invite.UserMonths, invite.UserDays).Add(time.Duration((60*invite.UserHours)+invite.UserMinutes) * time.Minute)
		app.storage.SetUserExpiryKey(nu.User.ID, UserExpiry{Expiry: expiry})
	}

	var discordUser *DiscordUser = nil
	var telegramUser *TelegramUser = nil
	if app.ombi.Enabled(app, profile) || app.js.Enabled(app, profile) {
		// FIXME: figure these out in a nicer way? this relies on the current ordering,
		// which may not be fixed.
		if discordEnabled {
			if req.completeContactMethods[0].User != nil {
				discordUser = req.completeContactMethods[0].User.(*DiscordUser)
			}
			if telegramEnabled && req.completeContactMethods[1].User != nil {
				telegramUser = req.completeContactMethods[1].User.(*TelegramUser)
			}
		} else if telegramEnabled && req.completeContactMethods[0].User != nil {
			telegramUser = req.completeContactMethods[0].User.(*TelegramUser)
		}
	}

	for _, tps := range app.thirdPartyServices {
		if !tps.Enabled(app, profile) {
			continue
		}
		// User already created, now we can link contact methods
		err := tps.AddContactMethods(nu.User.ID, req.newUserDTO, discordUser, telegramUser)
		if err != nil {
			app.err.Printf(lm.FailedSyncContactMethods, tps.Name(), err)
		}
	}

	app.WelcomeNewUser(nu.User, expiry)
}

// @Summary Enable/Disable a list of users, optionally notifying them why.
// @Produce json
// @Param enableDisableUserDTO body enableDisableUserDTO true "User enable/disable request object"
// @Success 200 {object} boolResponse
// @Failure 400 {object} stringResponse
// @Failure 500 {object} errorListDTO "List of errors"
// @Router /users/enable [post]
// @Security Bearer
// @tags Users
func (app *appContext) EnableDisableUsers(gc *gin.Context) {
	var req enableDisableUserDTO
	gc.BindJSON(&req)
	errors := errorListDTO{
		"GetUser":   map[string]string{},
		"SetPolicy": map[string]string{},
	}
	sendMail := messagesEnabled
	var msg *Message
	var err error
	if sendMail {
		if req.Enabled {
			msg, err = app.email.constructEnabled(req.Reason, app, false)
		} else {
			msg, err = app.email.constructDisabled(req.Reason, app, false)
		}
		if err != nil {
			app.err.Printf(lm.FailedConstructEnableDisableMessage, "?", err)
			sendMail = false
		}
	}
	activityType := ActivityDisabled
	if req.Enabled {
		activityType = ActivityEnabled
	}
	for _, userID := range req.Users {
		user, err := app.jf.UserByID(userID, false)
		if err != nil {
			errors["GetUser"][user.ID] = err.Error()
			app.err.Printf(lm.FailedGetUser, user.ID, lm.Jellyfin, err)
			continue
		}
		err, _, _ = app.SetUserDisabled(user, !req.Enabled)
		if err != nil {
			errors["SetPolicy"][user.ID] = err.Error()
			app.err.Printf(lm.FailedApplyTemplate, "policy", lm.Jellyfin, user.ID, err)
			continue
		}

		// Record activity
		app.storage.SetActivityKey(shortuuid.New(), Activity{
			Type:       activityType,
			UserID:     user.ID,
			SourceType: ActivityAdmin,
			Source:     gc.GetString("jfId"),
			Time:       time.Now(),
		}, gc, false)

		if sendMail && req.Notify {
			if err := app.sendByID(msg, user.ID); err != nil {
				app.err.Printf(lm.FailedSendEnableDisableMessage, user.ID, "?", err)
				continue
			}
		}
	}
	app.InvalidateUserCaches()
	if len(errors["GetUser"]) != 0 || len(errors["SetPolicy"]) != 0 {
		gc.JSON(500, errors)
		return
	}
	respondBool(200, true, gc)
}

// @Summary Delete a list of users, optionally notifying them why.
// @Produce json
// @Param deleteUserDTO body deleteUserDTO true "User deletion request object"
// @Success 200 {object} boolResponse
// @Failure 400 {object} stringResponse
// @Failure 500 {object} errorListDTO "List of errors"
// @Router /users [delete]
// @Security Bearer
// @tags Users
func (app *appContext) DeleteUsers(gc *gin.Context) {
	var req deleteUserDTO
	gc.BindJSON(&req)
	errors := map[string]string{}
	sendMail := messagesEnabled
	var msg *Message
	var err error
	if sendMail {
		msg, err = app.email.constructDeleted(req.Reason, app, false)
		if err != nil {
			app.err.Printf(lm.FailedConstructDeletionMessage, "?", err)
			sendMail = false
		}
	}
	for _, userID := range req.Users {
		user, err := app.jf.UserByID(userID, false)
		if err != nil {
			app.err.Printf(lm.FailedGetUser, user.ID, lm.Jellyfin, err)
			errors[userID] = err.Error()
		}

		deleted := false
		err, deleted = app.DeleteUser(user)
		if err != nil {
			if _, ok := errors[user.ID]; !ok {
				errors[user.ID] = err.Error() + " "
			} else {
				errors[user.ID] += err.Error() + " "
			}
		}

		if deleted {
			// Record activity
			app.storage.SetActivityKey(shortuuid.New(), Activity{
				Type:       ActivityDeletion,
				UserID:     userID,
				SourceType: ActivityAdmin,
				Source:     gc.GetString("jfId"),
				Value:      user.Name,
				Time:       time.Now(),
			}, gc, false)
		}

		// Contact details are stored separately and periodically removed,
		// putting this here is hoping the daemon doesn't beat us.
		if sendMail && req.Notify {
			if err := app.sendByID(msg, user.ID); err != nil {
				app.err.Printf(lm.FailedSendDeletionMessage, userID, "?", err)
			}
		}
	}
	app.InvalidateUserCaches()
	if len(errors) == len(req.Users) {
		respondBool(500, false, gc)
		app.err.Printf(lm.FailedDeleteUsers, lm.Jellyfin, errors[req.Users[0]])
		return
	} else if len(errors) != 0 {
		gc.JSON(500, errors)
		return
	}
	respondBool(200, true, gc)
}

// @Summary Extend time before the user(s) expiry, or create an expiry if it doesn't exist.
// @Produce json
// @Param extendExpiryDTO body extendExpiryDTO true "Extend expiry object"
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /users/extend [post]
// @tags Users
func (app *appContext) ExtendExpiry(gc *gin.Context) {
	var req extendExpiryDTO
	gc.BindJSON(&req)
	if req.Months <= 0 && req.Days <= 0 && req.Hours <= 0 && req.Minutes <= 0 && req.Timestamp <= 0 {
		respondBool(400, false, gc)
		return
	}
	for _, id := range req.Users {
		base := time.Now()
		if expiry, ok := app.storage.GetUserExpiryKey(id); ok {
			base = expiry.Expiry
		}
		app.debug.Printf(lm.ExtendCreateExpiry, id)
		expiry := UserExpiry{}
		if req.Timestamp != 0 {
			expiry.Expiry = time.Unix(req.Timestamp, 0)
		} else {
			expiry.Expiry = base.AddDate(0, req.Months, req.Days).Add(time.Duration(((60 * req.Hours) + req.Minutes)) * time.Minute)
		}
		app.storage.SetUserExpiryKey(id, expiry)
		if messagesEnabled && req.Notify {
			go func(uid string, exp time.Time) {
				user, err := app.jf.UserByID(uid, false)
				if err != nil {
					return
				}
				msg, err := app.email.constructExpiryAdjusted(user.Name, exp, req.Reason, app, false)
				if err != nil {
					app.err.Printf(lm.FailedConstructExpiryAdjustmentMessage, uid, err)
					return
				}
				if err := app.sendByID(msg, uid); err != nil {
					app.err.Printf(lm.FailedSendExpiryAdjustmentMessage, uid, "?", err)
				}
			}(id, expiry.Expiry)
		}
	}
	respondBool(204, true, gc)
}

// @Summary Remove an expiry from a user's account.
// @Produce json
// @Param id path string true "id of user to extend expiry of."
// @Success 200 {object} boolResponse
// @Router /users/{id}/expiry [delete]
// @tags Users
func (app *appContext) RemoveExpiry(gc *gin.Context) {
	app.storage.DeleteUserExpiryKey(gc.Param("id"))
	respondBool(200, true, gc)
}

// @Summary Enable referrals for the given user(s) based on the rules set in the given invite code, or profile.
// @Produce json
// @Param EnableDisableReferralDTO body EnableDisableReferralDTO true "List of users"
// @Param mode path string true "mode of template sourcing from 'invite' or 'profile'."
// @Param source path string true "invite code or profile name, depending on what mode is."
// @Param useExpiry path string true "with-expiry or none."
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /users/referral/{mode}/{source}/{useExpiry} [post]
// @Security Bearer
// @tags Users
func (app *appContext) EnableReferralForUsers(gc *gin.Context) {
	var req EnableDisableReferralDTO
	gc.BindJSON(&req)
	mode := gc.Param("mode")

	source := gc.Param("source")
	useExpiry := gc.Param("useExpiry") == "with-expiry"
	baseInv := Invite{}
	if mode == "profile" {
		profile, ok := app.storage.GetProfileKey(source)
		err := app.storage.db.Get(profile.ReferralTemplateKey, &baseInv)
		if !ok || profile.ReferralTemplateKey == "" || err != nil {
			app.debug.Printf(lm.FailedGetReferralTemplate, profile.ReferralTemplateKey, err)
			respondBool(400, false, gc)
			return

		}
		app.debug.Printf(lm.GetReferralTemplate, profile.ReferralTemplateKey)
	} else if mode == "invite" {
		// Get the invite, and modify it to turn it into a referral
		err := app.storage.db.Get(source, &baseInv)
		if err != nil {
			app.debug.Printf(lm.InvalidInviteCode, source)
			respondBool(400, false, gc)
			return
		}
	}
	for _, u := range req.Users {
		// 1. Wipe out any existing referral codes.
		app.storage.db.DeleteMatching(Invite{}, badgerhold.Where("ReferrerJellyfinID").Eq(u))

		// 2. Generate referral invite.
		inv := baseInv
		inv.Code = GenerateInviteCode()
		expiryDelta := inv.ValidTill.Sub(inv.Created)
		inv.Created = time.Now()
		if useExpiry {
			inv.ValidTill = inv.Created.Add(expiryDelta)
		} else {
			inv.ValidTill = inv.Created.Add(REFERRAL_EXPIRY_DAYS * 24 * time.Hour)
		}
		inv.IsReferral = true
		inv.ReferrerJellyfinID = u
		inv.UseReferralExpiry = useExpiry
		app.storage.SetInvitesKey(inv.Code, inv)
	}
}

// @Summary Disable referrals for the given user(s).
// @Produce json
// @Param EnableDisableReferralDTO body EnableDisableReferralDTO true "List of users"
// @Success 200 {object} boolResponse
// @Router /users/referral [delete]
// @Security Bearer
// @tags Users
func (app *appContext) DisableReferralForUsers(gc *gin.Context) {
	var req EnableDisableReferralDTO
	gc.BindJSON(&req)
	for _, u := range req.Users {
		// 1. Delete directly bound template
		app.storage.db.DeleteMatching(Invite{}, badgerhold.Where("ReferrerJellyfinID").Eq(u))
		// 2. Check for and delete profile-attached template
		user, ok := app.storage.GetEmailsKey(u)
		if !ok {
			continue
		}
		user.ReferralTemplateKey = ""
		app.storage.SetEmailsKey(u, user)
	}
	respondBool(200, true, gc)
}

// @Summary Send an announcement via email to a given list of users.
// @Produce json
// @Param announcementDTO body announcementDTO true "Announcement request object"
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /users/announce [post]
// @Security Bearer
// @tags Users
func (app *appContext) Announce(gc *gin.Context) {
	var req announcementDTO
	gc.BindJSON(&req)
	if !messagesEnabled {
		respondBool(400, false, gc)
		return
	}
	// Generally, we only need to construct once. If {username} is included, however, this needs to be done for each user.
	unique := strings.Contains(req.Message, "{username}")
	if unique {
		for _, userID := range req.Users {
			user, err := app.jf.UserByID(userID, false)
			if err != nil {
				app.err.Printf(lm.FailedGetUser, userID, lm.Jellyfin, err)
				continue
			}
			msg, err := app.email.constructTemplate(req.Subject, req.Message, app, user.Name)
			if err != nil {
				app.err.Printf(lm.FailedConstructAnnouncementMessage, userID, err)
				respondBool(500, false, gc)
				return
			} else if err := app.sendByID(msg, userID); err != nil {
				app.err.Printf(lm.FailedSendAnnouncementMessage, userID, "?", err)
				respondBool(500, false, gc)
				return
			}
		}
		// app.info.Printf(lm.SentAnnouncementMessage, "*", "?")
	} else {
		msg, err := app.email.constructTemplate(req.Subject, req.Message, app)
		if err != nil {
			app.err.Printf(lm.FailedConstructAnnouncementMessage, "*", err)
			respondBool(500, false, gc)
			return
		} else if err := app.sendByID(msg, req.Users...); err != nil {
			app.err.Printf(lm.FailedSendAnnouncementMessage, "*", "?", err)
			respondBool(500, false, gc)
			return
		}
		// app.info.Printf(lm.SentAnnouncementMessage, "*", "?")
	}
	app.info.Printf(lm.SentAnnouncementMessage, "*", "?")
	respondBool(200, true, gc)
}

// @Summary Save an announcement as a template for use or editing later.
// @Produce json
// @Param announcementTemplate body announcementTemplate true "Announcement request object"
// @Success 200 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /users/announce/template [post]
// @Security Bearer
// @tags Users
func (app *appContext) SaveAnnounceTemplate(gc *gin.Context) {
	var req announcementTemplate
	gc.BindJSON(&req)
	if !messagesEnabled {
		respondBool(400, false, gc)
		return
	}

	app.storage.SetAnnouncementsKey(req.Name, req)
	respondBool(200, true, gc)
}

// @Summary Gets the names of each available announcement template.
// @Produce json
// @Success 200 {object} getAnnouncementsDTO
// @Router /users/announce [get]
// @Security Bearer
// @tags Users
func (app *appContext) GetAnnounceTemplates(gc *gin.Context) {
	resp := &getAnnouncementsDTO{make([]string, len(app.storage.GetAnnouncements()))}
	for i, a := range app.storage.GetAnnouncements() {
		resp.Announcements[i] = a.Name
	}
	gc.JSON(200, resp)
}

// @Summary Get an announcement template.
// @Produce json
// @Success 200 {object} announcementTemplate
// @Failure 400 {object} boolResponse
// @Param name path string true "name of template (url encoded if necessary)"
// @Router /users/announce/template/{name} [get]
// @Security Bearer
// @tags Users
func (app *appContext) GetAnnounceTemplate(gc *gin.Context) {
	escapedName := gc.Param("name")
	name, err := url.QueryUnescape(escapedName)
	if err != nil {
		respondBool(400, false, gc)
		return
	}
	if announcement, ok := app.storage.GetAnnouncementsKey(name); ok {
		gc.JSON(200, announcement)
		return
	}
	respondBool(400, false, gc)
}

// @Summary Delete an announcement template.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param name path string true "name of template"
// @Router /users/announce/template/{name} [delete]
// @Security Bearer
// @tags Users
func (app *appContext) DeleteAnnounceTemplate(gc *gin.Context) {
	name := gc.Param("name")
	app.storage.DeleteAnnouncementsKey(name)
	respondBool(200, false, gc)
}

// @Summary Generate password reset links for a list of users, sending the links to them if possible.
// @Produce json
// @Param AdminPasswordResetDTO body AdminPasswordResetDTO true "List of user IDs"
// @Success 204 {object} boolResponse
// @Success 200 {object} AdminPasswordResetRespDTO
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /users/password-reset [post]
// @Security Bearer
// @tags Users
func (app *appContext) AdminPasswordReset(gc *gin.Context) {
	var req AdminPasswordResetDTO
	gc.BindJSON(&req)
	if req.Users == nil || len(req.Users) == 0 {
		respondBool(400, false, gc)
		return
	}
	linkCount := 0
	var pwr InternalPWR
	var err error
	resp := AdminPasswordResetRespDTO{}
	for _, id := range req.Users {
		pwr, err = app.GenInternalReset(id)
		if err != nil {
			app.err.Printf(lm.FailedGetUser, id, lm.Jellyfin, err)
			respondBool(500, false, gc)
			return
		}
		if app.internalPWRs == nil {
			app.internalPWRs = map[string]InternalPWR{}
		}
		app.internalPWRs[pwr.PIN] = pwr
		sendAddress := app.getAddressOrName(id)
		if sendAddress == "" || len(req.Users) == 1 {
			resp.Link, err = app.GenResetLink(pwr.PIN)
			linkCount++
			if sendAddress == "" {
				resp.Manual = true
			}
		}
		if sendAddress != "" {
			msg, err := app.email.constructReset(
				PasswordReset{
					Pin:      pwr.PIN,
					Username: pwr.Username,
					Expiry:   pwr.Expiry,
					Internal: true,
				}, app, false,
			)
			if err != nil {
				app.err.Printf(lm.FailedConstructPWRMessage, id, err)
				respondBool(500, false, gc)
				return
			} else if err := app.sendByID(msg, id); err != nil {
				app.err.Printf(lm.FailedSendPWRMessage, id, sendAddress, err)
			} else {
				app.info.Printf(lm.SentPWRMessage, id, sendAddress)
			}
		}
	}
	if resp.Link != "" && linkCount == 1 {
		gc.JSON(200, resp)
		return
	}
	respondBool(204, true, gc)
}

func (app *appContext) userSummary(jfUser mediabrowser.User) respUser {
	adminOnly := app.config.Section("ui").Key("admin_only").MustBool(true)
	allowAll := app.config.Section("ui").Key("allow_all").MustBool(false)
	referralsEnabled := app.config.Section("user_page").Key("referrals").MustBool(false)
	user := respUser{
		ID:               jfUser.ID,
		Name:             jfUser.Name,
		Admin:            jfUser.Policy.IsAdministrator,
		Disabled:         jfUser.Policy.IsDisabled,
		ReferralsEnabled: false,
	}
	if !jfUser.LastActivityDate.IsZero() {
		user.LastActive = jfUser.LastActivityDate.Unix()
	}
	if email, ok := app.storage.GetEmailsKey(jfUser.ID); ok {
		user.Email = email.Addr
		user.NotifyThroughEmail = email.Contact
		user.Label = email.Label
		user.AccountsAdmin = (app.jellyfinLogin) && (email.Admin || (adminOnly && jfUser.Policy.IsAdministrator) || allowAll)
	}
	expiry, ok := app.storage.GetUserExpiryKey(jfUser.ID)
	if ok {
		user.Expiry = expiry.Expiry.Unix()
	}
	if tgUser, ok := app.storage.GetTelegramKey(jfUser.ID); ok {
		user.Telegram = tgUser.Username
		user.NotifyThroughTelegram = tgUser.Contact
	}
	if mxUser, ok := app.storage.GetMatrixKey(jfUser.ID); ok {
		user.Matrix = mxUser.UserID
		user.NotifyThroughMatrix = mxUser.Contact
	}
	if dcUser, ok := app.storage.GetDiscordKey(jfUser.ID); ok {
		user.Discord = RenderDiscordUsername(dcUser)
		// user.Discord = dcUser.Username + "#" + dcUser.Discriminator
		user.DiscordID = dcUser.ID
		user.NotifyThroughDiscord = dcUser.Contact
	}
	// FIXME: Send referral data
	referrerInv := Invite{}
	if referralsEnabled {
		// 1. Directly attached invite.
		err := app.storage.db.FindOne(&referrerInv, badgerhold.Where("ReferrerJellyfinID").Eq(jfUser.ID))
		if err == nil {
			user.ReferralsEnabled = true
			// 2. Referrals via profile template. Shallow check, doesn't look for the thing in the database.
		} else if email, ok := app.storage.GetEmailsKey(jfUser.ID); ok && email.ReferralTemplateKey != "" {
			user.ReferralsEnabled = true
		}
	}
	return user

}

// @Summary Returns the total number of Jellyfin users.
// @Produce json
// @Success 200 {object} PageCountDTO
// @Router /users/count [get]
// @Security Bearer
// @tags Activity
func (app *appContext) GetUserCount(gc *gin.Context) {
	resp := PageCountDTO{}
	userList, err := app.userCache.GetUserDTOs(app, false)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	resp.Count = uint64(len(userList))
	gc.JSON(200, resp)
}

// @Summary Get a list of -all- Jellyfin users.
// @Produce json
// @Success 200 {object} getUsersDTO
// @Failure 500 {object} stringResponse
// @Router /users [get]
// @Security Bearer
// @tags Users
func (app *appContext) GetUsers(gc *gin.Context) {
	var resp getUsersDTO
	// We're sending all users, so this is always true
	resp.LastPage = true
	var err error
	resp.UserList, err = app.userCache.GetUserDTOs(app, true)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	gc.JSON(200, resp)
}

// @Summary Get a paginated, searchable list of Jellyfin users.
// @Produce json
// @Param ServerSearchReqDTO body ServerSearchReqDTO true "search / pagination parameters"
// @Success 200 {object} getUsersDTO
// @Failure 500 {object} stringResponse
// @Router /users [post]
// @Security Bearer
// @tags Users
func (app *appContext) SearchUsers(gc *gin.Context) {
	req := ServerSearchReqDTO{}
	gc.BindJSON(&req)
	if req.SortByField == "" {
		req.SortByField = USER_DEFAULT_SORT_FIELD
	}

	var resp getUsersDTO
	userList, err := app.userCache.GetUserDTOs(app, req.SortByField == USER_DEFAULT_SORT_FIELD)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	var filtered []*respUser
	if len(req.SearchTerms) != 0 || len(req.Queries) != 0 {
		filtered = app.userCache.Filter(userList, req.SearchTerms, req.Queries)
	} else {
		filtered = slices.Clone(userList)
	}

	if req.SortByField == USER_DEFAULT_SORT_FIELD {
		if req.Ascending != USER_DEFAULT_SORT_ASCENDING {
			slices.Reverse(filtered)
		}
	} else {
		app.userCache.Sort(filtered, req.SortByField, req.Ascending)
	}

	startIndex := (req.Page * req.Limit)
	if startIndex < len(filtered) {
		endIndex := min(startIndex+req.Limit, len(filtered))
		resp.UserList = filtered[startIndex:endIndex]
	}
	resp.LastPage = len(resp.UserList) != req.Limit
	gc.JSON(200, resp)
}

// @Summary Set whether or not a user can access jfa-go. Redundant if the user is a Jellyfin admin.
// @Produce json
// @Param setAccountsAdminDTO body setAccountsAdminDTO true "Map of userIDs to whether or not they have access."
// @Success 204 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /users/accounts-admin [post]
// @Security Bearer
// @tags Users
func (app *appContext) SetAccountsAdmin(gc *gin.Context) {
	var req setAccountsAdminDTO
	gc.BindJSON(&req)
	users, err := app.jf.GetUsers(false)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	for _, jfUser := range users {
		id := jfUser.ID
		if admin, ok := req[id]; ok {
			var emailStore = EmailAddress{}
			if oldEmail, ok := app.storage.GetEmailsKey(id); ok {
				emailStore = oldEmail
			}
			emailStore.Admin = admin
			app.storage.SetEmailsKey(id, emailStore)
			app.info.Printf(lm.UserAdminAdjusted, id, admin)
		}
	}
	respondBool(204, true, gc)
}

// @Summary Modify user's labels, which show next to their name in the accounts tab.
// @Produce json
// @Param modifyEmailsDTO body modifyEmailsDTO true "Map of userIDs to labels"
// @Success 204 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /users/labels [post]
// @Security Bearer
// @tags Users
func (app *appContext) ModifyLabels(gc *gin.Context) {
	var req modifyEmailsDTO
	gc.BindJSON(&req)
	users, err := app.jf.GetUsers(false)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	for _, jfUser := range users {
		id := jfUser.ID
		if label, ok := req[id]; ok {
			var emailStore = EmailAddress{}
			if oldEmail, ok := app.storage.GetEmailsKey(id); ok {
				emailStore = oldEmail
			}
			emailStore.Label = label
			app.debug.Printf(lm.UserLabelAdjusted, id, label)
			app.storage.SetEmailsKey(id, emailStore)
		}
	}
	respondBool(204, true, gc)
}

func (app *appContext) modifyEmail(jfID string, addr string) {
	contactPrefChanged := false
	emailStore, ok := app.storage.GetEmailsKey(jfID)
	// Auto enable contact by email for newly added addresses
	if !ok || emailStore.Addr == "" {
		emailStore = EmailAddress{
			Contact: true,
		}
		contactPrefChanged = true
	}
	emailStore.Addr = addr
	app.storage.SetEmailsKey(jfID, emailStore)
	if app.config.Section("ombi").Key("enabled").MustBool(false) {
		ombiUser, err := app.getOmbiUser(jfID)
		if err == nil {
			ombiUser["emailAddress"] = addr
			err = app.ombi.ModifyUser(ombiUser)
			if err != nil {
				app.err.Printf(lm.FailedSetEmailAddress, lm.Ombi, jfID, err)
			}
		}
	}
	if app.config.Section("jellyseerr").Key("enabled").MustBool(false) {
		err := app.js.ModifyMainUserSettings(jfID, jellyseerr.MainUserSettings{Email: addr})
		if err != nil {
			app.err.Printf(lm.FailedSetEmailAddress, lm.Jellyseerr, jfID, err)
		} else if contactPrefChanged {
			contactMethods := map[jellyseerr.NotificationsField]any{
				jellyseerr.FieldEmailEnabled: true,
			}
			err := app.js.ModifyNotifications(jfID, contactMethods)
			if err != nil {
				app.err.Printf(lm.FailedSyncContactMethods, lm.Jellyseerr, err)
			}
		}
	}
}

// @Summary Modify user's email addresses.
// @Produce json
// @Param modifyEmailsDTO body modifyEmailsDTO true "Map of userIDs to email addresses"
// @Success 200 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /users/emails [post]
// @Security Bearer
// @tags Users
func (app *appContext) ModifyEmails(gc *gin.Context) {
	var req modifyEmailsDTO
	gc.BindJSON(&req)
	users, err := app.jf.GetUsers(false)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	for _, jfUser := range users {
		id := jfUser.ID
		if address, ok := req[id]; ok {
			app.modifyEmail(id, address)

			app.info.Printf(lm.UserEmailAdjusted, gc.GetString("jfId"))

			activityType := ActivityContactLinked
			if address == "" {
				activityType = ActivityContactUnlinked
			}
			app.storage.SetActivityKey(shortuuid.New(), Activity{
				Type:       activityType,
				UserID:     id,
				SourceType: ActivityAdmin,
				Source:     gc.GetString("jfId"),
				Value:      "email",
				Time:       time.Now(),
			}, gc, false)
		}
	}
	respondBool(200, true, gc)
}

// @Summary Apply settings to a list of users, either from a profile or from another user.
// @Produce json
// @Param userSettingsDTO body userSettingsDTO true "Parameters for applying settings"
// @Success 200 {object} errorListDTO
// @Failure 500 {object} errorListDTO "Lists of errors that occurred while applying settings"
// @Router /users/settings [post]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) ApplySettings(gc *gin.Context) {
	app.info.Println("User settings change requested")
	var req userSettingsDTO
	gc.BindJSON(&req)
	applyingFromType := lm.Profile
	applyingFromSource := "?"
	var policy mediabrowser.Policy
	var configuration mediabrowser.Configuration
	var displayprefs map[string]interface{}
	var ombi map[string]interface{}
	var jellyseerr JellyseerrTemplate
	jellyseerr.Enabled = false
	if req.From == "profile" {
		// Check profile exists & isn't empty
		profile, ok := app.storage.GetProfileKey(req.Profile)
		if !ok {
			app.err.Printf(lm.FailedGetProfile, req.Profile)
			respond(500, "Couldn't find profile", gc)
			return
		}
		applyingFromSource = req.Profile
		if req.Homescreen {
			if profile.Homescreen {
				configuration = profile.Configuration
				displayprefs = profile.Displayprefs
			} else {
				req.Homescreen = false
				app.err.Printf(lm.ProfileNoHomescreen, req.Profile)
				respond(500, "No homescreen template available", gc)
				return
			}
		}
		if req.Policy {
			policy = profile.Policy
		}
		if req.Ombi && app.config.Section("ombi").Key("enabled").MustBool(false) {
			if profile.Ombi != nil && len(profile.Ombi) != 0 {
				ombi = profile.Ombi
			}
		}
		if req.Jellyseerr && app.config.Section("jellyseerr").Key("enabled").MustBool(false) {
			if profile.Jellyseerr.Enabled {
				jellyseerr = profile.Jellyseerr
			}
		}

	} else if req.From == "user" {
		applyingFromType = lm.User
		app.InvalidateJellyfinCache()
		user, err := app.jf.UserByID(req.ID, false)
		if err != nil {
			app.err.Printf(lm.FailedGetUser, req.ID, lm.Jellyfin, err)
			respond(500, "Couldn't get user", gc)
			return
		}
		applyingFromSource = user.Name
		if req.Policy {
			policy = user.Policy
		}
		if req.Homescreen {
			displayprefs, err = app.jf.GetDisplayPreferences(req.ID)
			if err != nil {
				app.err.Printf(lm.FailedGetJellyfinDisplayPrefs, req.ID, err)
				respond(500, "Couldn't get displayprefs", gc)
				return
			}
			configuration = user.Configuration
		}
	}
	app.info.Printf(lm.ApplyingTemplatesFrom, applyingFromType, applyingFromSource, len(req.ApplyTo))
	errors := errorListDTO{
		"policy":     map[string]string{},
		"homescreen": map[string]string{},
		"ombi":       map[string]string{},
		"jellyseerr": map[string]string{},
	}
	/* Jellyfin doesn't seem to like too many of these requests sent in succession
	and can crash and mess up its database. Issue #160 says this occurs when more
	than 100 users are modified. A delay totalling 500ms between requests is used
	if so. */
	const requestDelayThreshold = 100
	var shouldDelay bool = len(req.ApplyTo) >= requestDelayThreshold
	if shouldDelay {
		app.debug.Printf(lm.DelayingRequests, requestDelayThreshold)
	}
	for _, id := range req.ApplyTo {
		var err error
		if req.Policy {
			err = app.jf.SetPolicy(id, policy)
			if err != nil {
				errors["policy"][id] = err.Error()
			}
		}
		if shouldDelay {
			time.Sleep(250 * time.Millisecond)
		}
		if req.Homescreen {
			err = app.jf.SetConfiguration(id, configuration)
			errorString := ""
			if err != nil {
				errorString += fmt.Sprintf("Configuration: %v", err)
			} else {
				err = app.jf.SetDisplayPreferences(id, displayprefs)
				if err != nil {
					errorString += fmt.Sprintf("Displayprefs;) %v ", err)
				}
			}
			if errorString != "" {
				errors["homescreen"][id] = errorString
			}
		}
		if ombi != nil {
			errorString := ""
			user, err := app.getOmbiUser(id)
			if err != nil {
				errorString += fmt.Sprintf("Ombi GetUser: %v ", err)
			} else {
				// newUser := ombi
				// newUser["id"] = user["id"]
				// newUser["userName"] = user["userName"]
				// newUser["alias"] = user["alias"]
				// newUser["emailAddress"] = user["emailAddress"]
				err = app.ombi.applyProfile(user, ombi)
				if err != nil {
					errorString += fmt.Sprintf("Apply: %v ", err)
				}
			}
			if errorString != "" {
				errors["ombi"][id] = errorString
			}
		}
		if jellyseerr.Enabled {
			errorString := ""
			// newUser := ombi
			// newUser["id"] = user["id"]
			// newUser["userName"] = user["userName"]
			// newUser["alias"] = user["alias"]
			// newUser["emailAddress"] = user["emailAddress"]
			err := app.js.ApplyTemplateToUser(id, jellyseerr.User)
			if err != nil {
				errorString += fmt.Sprintf("ApplyUser: %v ", err)
			}
			err = app.js.ApplyNotificationsTemplateToUser(id, jellyseerr.Notifications)
			if err != nil {
				errorString += fmt.Sprintf("ApplyNotifications: %v ", err)
			}
			if errorString != "" {
				errors["jellyseerr"][id] = errorString
			}
		}

		if shouldDelay {
			time.Sleep(250 * time.Millisecond)
		}
	}
	code := 200
	if len(errors["policy"]) == len(req.ApplyTo) || len(errors["homescreen"]) == len(req.ApplyTo) {
		code = 500
	}
	gc.JSON(code, errors)
}
