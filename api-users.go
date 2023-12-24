package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/hrfee/mediabrowser"
	"github.com/lithammer/shortuuid/v3"
	"github.com/timshannon/badgerhold/v4"
)

// @Summary Creates a new Jellyfin user without an invite.
// @Produce json
// @Param newUserDTO body newUserDTO true "New user request object"
// @Success 200
// @Router /users [post]
// @Security Bearer
// @tags Users
func (app *appContext) NewUserAdmin(gc *gin.Context) {
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
	existingUser, _, _ := app.jf.UserByName(req.Username, false)
	if existingUser.Name != "" {
		msg := fmt.Sprintf("User already exists named %s", req.Username)
		app.info.Printf("%s New user failed: %s", req.Username, msg)
		respondUser(401, false, false, msg, gc)
		return
	}
	user, status, err := app.jf.NewUser(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("%s New user failed (%d): %v", req.Username, status, err)
		respondUser(401, false, false, err.Error(), gc)
		return
	}
	id := user.ID

	// Record activity
	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityCreation,
		UserID:     id,
		SourceType: ActivityAdmin,
		Source:     gc.GetString("jfId"),
		Value:      user.Name,
		Time:       time.Now(),
	}, gc, false)

	profile := app.storage.GetDefaultProfile()
	if req.Profile != "" && req.Profile != "none" {
		if p, ok := app.storage.GetProfileKey(req.Profile); ok {
			profile = p
		} else {
			app.debug.Printf("Couldn't find profile \"%s\", using default", req.Profile)
		}

		status, err = app.jf.SetPolicy(id, profile.Policy)
		if !(status == 200 || status == 204 || err == nil) {
			app.err.Printf("%s: Failed to set user policy (%d): %v", req.Username, status, err)
		}
		status, err = app.jf.SetConfiguration(id, profile.Configuration)
		if (status == 200 || status == 204) && err == nil {
			status, err = app.jf.SetDisplayPreferences(id, profile.Displayprefs)
		}
		if !((status == 200 || status == 204) && err == nil) {
			app.err.Printf("%s: Failed to set configuration template (%d): %v", req.Username, status, err)
		}
	}
	app.jf.CacheExpiry = time.Now()
	if emailEnabled {
		app.storage.SetEmailsKey(id, EmailAddress{Addr: req.Email, Contact: true})
	}
	if app.config.Section("ombi").Key("enabled").MustBool(false) {
		if profile.Ombi == nil {
			profile.Ombi = map[string]interface{}{}
		}
		errors, code, err := app.ombi.NewUser(req.Username, req.Password, req.Email, profile.Ombi)
		if err != nil || code != 200 {
			app.err.Printf("Failed to create Ombi user (%d): %v", code, err)
			app.debug.Printf("Errors reported by Ombi: %s", strings.Join(errors, ", "))
		} else {
			app.info.Println("Created Ombi user")
		}
	}
	if emailEnabled && app.config.Section("welcome_email").Key("enabled").MustBool(false) && req.Email != "" {
		app.debug.Printf("%s: Sending welcome email to %s", req.Username, req.Email)
		msg, err := app.email.constructWelcome(req.Username, time.Time{}, app, false)
		if err != nil {
			app.err.Printf("%s: Failed to construct welcome email: %v", req.Username, err)
			respondUser(500, true, false, err.Error(), gc)
			return
		} else if err := app.email.send(msg, req.Email); err != nil {
			app.err.Printf("%s: Failed to send welcome email: %v", req.Username, err)
			respondUser(500, true, false, err.Error(), gc)
			return
		} else {
			app.info.Printf("%s: Sent welcome email to %s", req.Username, req.Email)
		}
	}
	respondUser(200, true, true, "", gc)
}

type errorFunc func(gc *gin.Context)

// Used on the form & when a users email has been confirmed.
func (app *appContext) newUser(req newUserDTO, confirmed bool, gc *gin.Context) (f errorFunc, success bool) {
	existingUser, _, _ := app.jf.UserByName(req.Username, false)
	if existingUser.Name != "" {
		f = func(gc *gin.Context) {
			msg := fmt.Sprintf("User %s already exists", req.Username)
			app.info.Printf("%s: New user failed: %s", req.Code, msg)
			respond(401, "errorUserExists", gc)
		}
		success = false
		return
	}
	var discordUser DiscordUser
	discordVerified := false
	if discordEnabled {
		if req.DiscordPIN == "" {
			if app.config.Section("discord").Key("required").MustBool(false) {
				f = func(gc *gin.Context) {
					app.debug.Printf("%s: New user failed: Discord verification not completed", req.Code)
					respond(401, "errorDiscordVerification", gc)
				}
				success = false
				return
			}
		} else {
			discordUser, discordVerified = app.discord.UserVerified(req.DiscordPIN)
			if !discordVerified {
				f = func(gc *gin.Context) {
					app.debug.Printf("%s: New user failed: Discord PIN was invalid", req.Code)
					respond(401, "errorInvalidPIN", gc)
				}
				success = false
				return
			}
			if app.config.Section("discord").Key("require_unique").MustBool(false) && app.discord.UserExists(discordUser.ID) {
				f = func(gc *gin.Context) {
					app.debug.Printf("%s: New user failed: Discord user already linked", req.Code)
					respond(400, "errorAccountLinked", gc)
				}
				success = false
				return
			}
			err := app.discord.ApplyRole(discordUser.ID)
			if err != nil {
				f = func(gc *gin.Context) {
					app.err.Printf("%s: New user failed: Failed to set member role: %v", req.Code, err)
					respond(401, "error", gc)
				}
				success = false
				return
			}
		}
	}
	var matrixUser MatrixUser
	matrixVerified := false
	if matrixEnabled {
		if req.MatrixPIN == "" {
			if app.config.Section("matrix").Key("required").MustBool(false) {
				f = func(gc *gin.Context) {
					app.debug.Printf("%s: New user failed: Matrix verification not completed", req.Code)
					respond(401, "errorMatrixVerification", gc)
				}
				success = false
				return
			}
		} else {
			user, ok := app.matrix.tokens[req.MatrixPIN]
			if !ok || !user.Verified {
				matrixVerified = false
				f = func(gc *gin.Context) {
					app.debug.Printf("%s: New user failed: Matrix PIN was invalid", req.Code)
					respond(401, "errorInvalidPIN", gc)
				}
				success = false
				return
			}
			if app.config.Section("matrix").Key("require_unique").MustBool(false) && app.matrix.UserExists(user.User.UserID) {
				f = func(gc *gin.Context) {
					app.debug.Printf("%s: New user failed: Matrix user already linked", req.Code)
					respond(400, "errorAccountLinked", gc)
				}
				success = false
				return
			}
			matrixVerified = user.Verified
			matrixUser = *user.User

		}
	}
	var tgToken TelegramVerifiedToken
	telegramVerified := false
	if telegramEnabled {
		if req.TelegramPIN == "" {
			if app.config.Section("telegram").Key("required").MustBool(false) {
				f = func(gc *gin.Context) {
					app.debug.Printf("%s: New user failed: Telegram verification not completed", req.Code)
					respond(401, "errorTelegramVerification", gc)
				}
				success = false
				return
			}
		} else {
			tgToken, telegramVerified = app.telegram.TokenVerified(req.TelegramPIN)
			if !telegramVerified {
				f = func(gc *gin.Context) {
					app.debug.Printf("%s: New user failed: Telegram PIN was invalid", req.Code)
					respond(401, "errorInvalidPIN", gc)
				}
				success = false
				return
			}
			if app.config.Section("telegram").Key("require_unique").MustBool(false) && app.telegram.UserExists(tgToken.Username) {
				f = func(gc *gin.Context) {
					app.debug.Printf("%s: New user failed: Telegram user already linked", req.Code)
					respond(400, "errorAccountLinked", gc)
				}
				success = false
				return
			}
		}
	}
	if emailEnabled && app.config.Section("email_confirmation").Key("enabled").MustBool(false) && !confirmed {
		claims := jwt.MapClaims{
			"valid":  true,
			"invite": req.Code,
			"exp":    time.Now().Add(30 * time.Minute).Unix(),
			"type":   "confirmation",
		}
		tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		key, err := tk.SignedString([]byte(os.Getenv("JFA_SECRET")))
		if err != nil {
			f = func(gc *gin.Context) {
				app.info.Printf("Failed to generate confirmation token: %v", err)
				respond(500, "errorUnknown", gc)
			}
			success = false
			return
		}
		if app.ConfirmationKeys == nil {
			app.ConfirmationKeys = map[string]map[string]newUserDTO{}
		}
		cKeys, ok := app.ConfirmationKeys[req.Code]
		if !ok {
			cKeys = map[string]newUserDTO{}
		}
		cKeys[key] = req
		app.confirmationKeysLock.Lock()
		app.ConfirmationKeys[req.Code] = cKeys
		app.confirmationKeysLock.Unlock()
		f = func(gc *gin.Context) {
			app.debug.Printf("%s: Email confirmation required", req.Code)
			respond(401, "confirmEmail", gc)
			msg, err := app.email.constructConfirmation(req.Code, req.Username, key, app, false)
			if err != nil {
				app.err.Printf("%s: Failed to construct confirmation email: %v", req.Code, err)
			} else if err := app.email.send(msg, req.Email); err != nil {
				app.err.Printf("%s: Failed to send user confirmation email: %v", req.Code, err)
			} else {
				app.info.Printf("%s: Sent user confirmation email to \"%s\"", req.Code, req.Email)
			}
		}
		success = false
		return
	}

	user, status, err := app.jf.NewUser(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		f = func(gc *gin.Context) {
			app.err.Printf("%s New user failed (%d): %v", req.Code, status, err)
			respond(401, app.storage.lang.Admin[app.storage.lang.chosenAdminLang].Notifications.get("errorUnknown"), gc)
		}
		success = false
		return
	}
	invite, _ := app.storage.GetInvitesKey(req.Code)
	app.checkInvite(req.Code, true, req.Username)
	if emailEnabled && app.config.Section("notifications").Key("enabled").MustBool(false) {
		for address, settings := range invite.Notify {
			if settings["notify-creation"] {
				go func() {
					msg, err := app.email.constructCreated(req.Code, req.Username, req.Email, invite, app, false)
					if err != nil {
						app.err.Printf("%s: Failed to construct user creation notification: %v", req.Code, err)
					} else {
						// Check whether notify "address" is an email address of Jellyfin ID
						if strings.Contains(address, "@") {
							err = app.email.send(msg, address)
						} else {
							err = app.sendByID(msg, address)
						}
						if err != nil {
							app.err.Printf("%s: Failed to send user creation notification: %v", req.Code, err)
						} else {
							app.info.Printf("Sent user creation notification to %s", address)
						}
					}
				}()
			}
		}
	}
	id := user.ID

	// Record activity
	sourceType := ActivityAnon
	source := ""
	if invite.ReferrerJellyfinID != "" {
		sourceType = ActivityUser
		source = invite.ReferrerJellyfinID
	}

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityCreation,
		UserID:     id,
		SourceType: sourceType,
		Source:     source,
		InviteCode: invite.Code,
		Value:      user.Name,
		Time:       time.Now(),
	}, gc, true)

	emailStore := EmailAddress{
		Addr:    req.Email,
		Contact: (req.Email != ""),
	}

	if invite.UserLabel != "" {
		emailStore.Label = invite.UserLabel
	}

	var profile Profile
	if invite.Profile != "" {
		app.debug.Printf("Applying settings from profile \"%s\"", invite.Profile)
		var ok bool
		profile, ok = app.storage.GetProfileKey(invite.Profile)
		if !ok {
			profile = app.storage.GetDefaultProfile()
		}
		app.debug.Printf("Applying policy from profile \"%s\"", invite.Profile)
		status, err = app.jf.SetPolicy(id, profile.Policy)
		if !((status == 200 || status == 204) && err == nil) {
			app.err.Printf("%s: Failed to set user policy (%d): %v", req.Code, status, err)
		}
		app.debug.Printf("Applying homescreen from profile \"%s\"", invite.Profile)
		status, err = app.jf.SetConfiguration(id, profile.Configuration)
		if (status == 200 || status == 204) && err == nil {
			status, err = app.jf.SetDisplayPreferences(id, profile.Displayprefs)
		}
		if !((status == 200 || status == 204) && err == nil) {
			app.err.Printf("%s: Failed to set configuration template (%d): %v", req.Code, status, err)
		}
		if app.config.Section("user_page").Key("enabled").MustBool(false) && app.config.Section("user_page").Key("referrals").MustBool(false) && profile.ReferralTemplateKey != "" {
			emailStore.ReferralTemplateKey = profile.ReferralTemplateKey
			// Store here, just incase email are disabled (whether this is even possible, i don't know)
			app.storage.SetEmailsKey(id, emailStore)

			// If UseReferralExpiry is enabled, create the ref now so the clock starts ticking
			refInv := Invite{}
			err = app.storage.db.Get(profile.ReferralTemplateKey, &refInv)
			if refInv.UseReferralExpiry {
				refInv.Code = GenerateInviteCode()
				expiryDelta := refInv.ValidTill.Sub(refInv.Created)
				refInv.Created = time.Now()
				refInv.ValidTill = refInv.Created.Add(expiryDelta)
				refInv.IsReferral = true
				refInv.ReferrerJellyfinID = id
				app.storage.SetInvitesKey(refInv.Code, refInv)
			}
		}
	}
	// if app.config.Section("password_resets").Key("enabled").MustBool(false) {
	if req.Email != "" || invite.UserLabel != "" {
		app.storage.SetEmailsKey(id, emailStore)
	}
	expiry := time.Time{}
	if invite.UserExpiry {
		expiry = time.Now().AddDate(0, invite.UserMonths, invite.UserDays).Add(time.Duration((60*invite.UserHours)+invite.UserMinutes) * time.Minute)
		app.storage.SetUserExpiryKey(id, UserExpiry{Expiry: expiry})
	}
	if discordVerified {
		discordUser.Contact = req.DiscordContact
		if app.storage.deprecatedDiscord == nil {
			app.storage.deprecatedDiscord = discordStore{}
		}
		// Note we don't log an activity here, since it's part of creating a user.
		app.storage.SetDiscordKey(user.ID, discordUser)
		delete(app.discord.verifiedTokens, req.DiscordPIN)
	}
	if telegramVerified {
		tgUser := TelegramUser{
			ChatID:   tgToken.ChatID,
			Username: tgToken.Username,
			Contact:  req.TelegramContact,
		}
		if lang, ok := app.telegram.languages[tgToken.ChatID]; ok {
			tgUser.Lang = lang
		}
		if app.storage.deprecatedTelegram == nil {
			app.storage.deprecatedTelegram = telegramStore{}
		}
		app.telegram.DeleteVerifiedToken(req.TelegramPIN)
		app.storage.SetTelegramKey(user.ID, tgUser)
	}
	if invite.Profile != "" && app.config.Section("ombi").Key("enabled").MustBool(false) {
		if profile.Ombi != nil && len(profile.Ombi) != 0 {
			template := profile.Ombi
			errors, code, err := app.ombi.NewUser(req.Username, req.Password, req.Email, template)
			accountExists := false
			var ombiUser map[string]interface{}
			if err != nil || code != 200 {
				// Check if on the off chance, Ombi's user importer has already added the account.
				ombiUser, status, err = app.getOmbiImportedUser(req.Username)
				if status == 200 && err == nil {
					app.info.Println("Found existing Ombi user, applying changes")
					accountExists = true
					template["password"] = req.Password
					status, err = app.applyOmbiProfile(ombiUser, template)
					if status != 200 || err != nil {
						app.err.Printf("Failed to modify existing Ombi user (%d): %v\n", status, err)
					}
				} else {
					app.info.Printf("Failed to create Ombi user (%d): %s", code, err)
					app.debug.Printf("Errors reported by Ombi: %s", strings.Join(errors, ", "))
				}
			} else {
				ombiUser, status, err = app.getOmbiUser(id)
				if status != 200 || err != nil {
					app.err.Printf("Failed to get Ombi user (%d): %v", status, err)
				} else {
					app.info.Println("Created Ombi user")
					accountExists = true
				}
			}
			if accountExists {
				if discordVerified || telegramVerified {
					dID := ""
					tUser := ""
					if discordVerified {
						dID = discordUser.ID
					}
					if telegramVerified {
						u, _ := app.storage.GetTelegramKey(user.ID)
						tUser = u.Username
					}
					resp, status, err := app.ombi.SetNotificationPrefs(ombiUser, dID, tUser)
					if !(status == 200 || status == 204) || err != nil {
						app.err.Printf("Failed to link Telegram/Discord to Ombi (%d): %v", status, err)
						app.debug.Printf("Response: %v", resp)
					}
				}
			}
		} else {
			app.debug.Printf("Skipping Ombi: Profile \"%s\" was empty", invite.Profile)
		}
	}
	if matrixVerified {
		matrixUser.Contact = req.MatrixContact
		delete(app.matrix.tokens, req.MatrixPIN)
		if app.storage.deprecatedMatrix == nil {
			app.storage.deprecatedMatrix = matrixStore{}
		}
		app.storage.SetMatrixKey(user.ID, matrixUser)
	}
	if (emailEnabled && app.config.Section("welcome_email").Key("enabled").MustBool(false) && req.Email != "") || telegramVerified || discordVerified || matrixVerified {
		name := app.getAddressOrName(user.ID)
		app.debug.Printf("%s: Sending welcome message to %s", req.Username, name)
		msg, err := app.email.constructWelcome(req.Username, expiry, app, false)
		if err != nil {
			app.err.Printf("%s: Failed to construct welcome message: %v", req.Username, err)
		} else if err := app.sendByID(msg, user.ID); err != nil {
			app.err.Printf("%s: Failed to send welcome message: %v", req.Username, err)
		} else {
			app.info.Printf("%s: Sent welcome message to \"%s\"", req.Username, name)
		}
	}
	app.jf.CacheExpiry = time.Now()
	success = true
	return
}

// @Summary Creates a new Jellyfin user via invite code
// @Produce json
// @Param newUserDTO body newUserDTO true "New user request object"
// @Success 200 {object} PasswordValidation
// @Failure 400 {object} PasswordValidation
// @Router /newUser [post]
// @tags Users
func (app *appContext) NewUser(gc *gin.Context) {
	var req newUserDTO
	gc.BindJSON(&req)
	app.debug.Printf("%s: New user attempt", req.Code)
	if app.config.Section("captcha").Key("enabled").MustBool(false) && !app.verifyCaptcha(req.Code, req.CaptchaID, req.CaptchaText, false) {
		app.info.Printf("%s: New user failed: Captcha Incorrect", req.Code)
		respond(400, "errorCaptcha", gc)
		return
	}
	if !app.checkInvite(req.Code, false, "") {
		app.info.Printf("%s New user failed: invalid code", req.Code)
		respond(401, "errorInvalidCode", gc)
		return
	}
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
		app.info.Printf("%s: New user failed: Invalid password", req.Code)
		gc.JSON(200, validation)
		return
	}
	if emailEnabled {
		if app.config.Section("email").Key("required").MustBool(false) && !strings.Contains(req.Email, "@") {
			app.info.Printf("%s: New user failed: Email Required", req.Code)
			respond(400, "errorNoEmail", gc)
			return
		}
		if app.config.Section("email").Key("require_unique").MustBool(false) && req.Email != "" && app.EmailAddressExists(req.Email) {
			app.info.Printf("%s: New user failed: Email already in use", req.Code)
			respond(400, "errorEmailLinked", gc)
			return
		}
	}
	f, success := app.newUser(req, false, gc)
	if !success {
		f(gc)
		return
	}
	code := 200
	for _, val := range validation {
		if !val {
			code = 400
		}
	}
	gc.JSON(code, validation)
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
			app.err.Printf("Failed to construct account enabled/disabled emails: %v", err)
			sendMail = false
		}
	}
	activityType := ActivityDisabled
	if req.Enabled {
		activityType = ActivityEnabled
	}
	for _, userID := range req.Users {
		user, status, err := app.jf.UserByID(userID, false)
		if status != 200 || err != nil {
			errors["GetUser"][userID] = fmt.Sprintf("%d %v", status, err)
			app.err.Printf("Failed to get user \"%s\" (%d): %v", userID, status, err)
			continue
		}
		user.Policy.IsDisabled = !req.Enabled
		status, err = app.jf.SetPolicy(userID, user.Policy)
		if !(status == 200 || status == 204) || err != nil {
			errors["SetPolicy"][userID] = fmt.Sprintf("%d %v", status, err)
			app.err.Printf("Failed to set policy for user \"%s\" (%d): %v", userID, status, err)
			continue
		}

		// Record activity
		app.storage.SetActivityKey(shortuuid.New(), Activity{
			Type:       activityType,
			UserID:     userID,
			SourceType: ActivityAdmin,
			Source:     gc.GetString("jfId"),
			Time:       time.Now(),
		}, gc, false)

		if sendMail && req.Notify {
			if err := app.sendByID(msg, userID); err != nil {
				app.err.Printf("Failed to send account enabled/disabled email: %v", err)
				continue
			}
		}
	}
	app.jf.CacheExpiry = time.Now()
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
	ombiEnabled := app.config.Section("ombi").Key("enabled").MustBool(false)
	sendMail := messagesEnabled
	var msg *Message
	var err error
	if sendMail {
		msg, err = app.email.constructDeleted(req.Reason, app, false)
		if err != nil {
			app.err.Printf("Failed to construct account deletion emails: %v", err)
			sendMail = false
		}
	}
	for _, userID := range req.Users {
		if ombiEnabled {
			ombiUser, code, err := app.getOmbiUser(userID)
			if code == 200 && err == nil {
				if id, ok := ombiUser["id"]; ok {
					status, err := app.ombi.DeleteUser(id.(string))
					if err != nil || status != 200 {
						app.err.Printf("Failed to delete ombi user (%d): %v", status, err)
						errors[userID] = fmt.Sprintf("Ombi: %d %v, ", status, err)
					}
				}
			}
		}

		username := ""
		if user, status, err := app.jf.UserByID(userID, false); status == 200 && err == nil {
			username = user.Name
		}

		status, err := app.jf.DeleteUser(userID)
		if !(status == 200 || status == 204) || err != nil {
			msg := fmt.Sprintf("%d: %v", status, err)
			if _, ok := errors[userID]; !ok {
				errors[userID] = msg
			} else {
				errors[userID] += msg
			}
		}

		// Record activity
		app.storage.SetActivityKey(shortuuid.New(), Activity{
			Type:       ActivityDeletion,
			UserID:     userID,
			SourceType: ActivityAdmin,
			Source:     gc.GetString("jfId"),
			Value:      username,
			Time:       time.Now(),
		}, gc, false)

		if sendMail && req.Notify {
			if err := app.sendByID(msg, userID); err != nil {
				app.err.Printf("Failed to send account deletion email: %v", err)
			}
		}
	}
	app.jf.CacheExpiry = time.Now()
	if len(errors) == len(req.Users) {
		respondBool(500, false, gc)
		app.err.Printf("Account deletion failed: %s", errors[req.Users[0]])
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
	app.info.Printf("Expiry extension requested for %d user(s)", len(req.Users))
	if req.Months <= 0 && req.Days <= 0 && req.Hours <= 0 && req.Minutes <= 0 && req.Timestamp <= 0 {
		respondBool(400, false, gc)
		return
	}
	for _, id := range req.Users {
		base := time.Now()
		if expiry, ok := app.storage.GetUserExpiryKey(id); ok {
			base = expiry.Expiry
			app.debug.Printf("Expiry extended for \"%s\"", id)
		} else {
			app.debug.Printf("Created expiry for \"%s\"", id)
		}
		expiry := UserExpiry{}
		if req.Timestamp != 0 {
			expiry.Expiry = time.Unix(req.Timestamp, 0)
		} else {
			expiry.Expiry = base.AddDate(0, req.Months, req.Days).Add(time.Duration(((60 * req.Hours) + req.Minutes)) * time.Minute)
		}
		app.storage.SetUserExpiryKey(id, expiry)
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
			app.debug.Printf("Couldn't find template to source from")
			respondBool(400, false, gc)
			return

		}
		app.debug.Printf("Found referral template in profile: %+v\n", profile.ReferralTemplateKey)
	} else if mode == "invite" {
		// Get the invite, and modify it to turn it into a referral
		err := app.storage.db.Get(source, &baseInv)
		if err != nil {
			app.debug.Printf("Couldn't find invite to source from")
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
			user, status, err := app.jf.UserByID(userID, false)
			if status != 200 || err != nil {
				app.err.Printf("Failed to get user with ID \"%s\" (%d): %v", userID, status, err)
				continue
			}
			msg, err := app.email.constructTemplate(req.Subject, req.Message, app, user.Name)
			if err != nil {
				app.err.Printf("Failed to construct announcement message: %v", err)
				respondBool(500, false, gc)
				return
			} else if err := app.sendByID(msg, userID); err != nil {
				app.err.Printf("Failed to send announcement message: %v", err)
				respondBool(500, false, gc)
				return
			}
		}
	} else {
		msg, err := app.email.constructTemplate(req.Subject, req.Message, app)
		if err != nil {
			app.err.Printf("Failed to construct announcement messages: %v", err)
			respondBool(500, false, gc)
			return
		} else if err := app.sendByID(msg, req.Users...); err != nil {
			app.err.Printf("Failed to send announcement messages: %v", err)
			respondBool(500, false, gc)
			return
		}
	}
	app.info.Println("Sent announcement messages")
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

// @Summary Save an announcement as a template for use or editing later.
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
// @Param name path string true "name of template"
// @Router /users/announce/template/{name} [get]
// @Security Bearer
// @tags Users
func (app *appContext) GetAnnounceTemplate(gc *gin.Context) {
	name := gc.Param("name")
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
		app.debug.Println("Ignoring empty request for PWR")
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
			app.err.Printf("Failed to get user from Jellyfin: %v", err)
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
				app.err.Printf("Failed to construct password reset message for \"%s\": %v", pwr.Username, err)
				respondBool(500, false, gc)
				return
			} else if err := app.sendByID(msg, id); err != nil {
				app.err.Printf("Failed to send password reset message to \"%s\": %v", sendAddress, err)
			} else {
				app.info.Printf("Sent password reset message to \"%s\"", sendAddress)
			}
		}
	}
	if resp.Link != "" && linkCount == 1 {
		gc.JSON(200, resp)
		return
	}
	respondBool(204, true, gc)
}

// @Summary Get a list of Jellyfin users.
// @Produce json
// @Success 200 {object} getUsersDTO
// @Failure 500 {object} stringResponse
// @Router /users [get]
// @Security Bearer
// @tags Users
func (app *appContext) GetUsers(gc *gin.Context) {
	app.debug.Println("Users requested")
	var resp getUsersDTO
	users, status, err := app.jf.GetUsers(false)
	resp.UserList = make([]respUser, len(users))
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	adminOnly := app.config.Section("ui").Key("admin_only").MustBool(true)
	allowAll := app.config.Section("ui").Key("allow_all").MustBool(false)
	referralsEnabled := app.config.Section("user_page").Key("referrals").MustBool(false)
	i := 0
	for _, jfUser := range users {
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
		resp.UserList[i] = user
		i++
	}
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
	app.debug.Println("Admin modification requested")
	users, status, err := app.jf.GetUsers(false)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
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
		}
	}
	app.info.Println("Email list modified")
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
	app.debug.Println("Label modification requested")
	users, status, err := app.jf.GetUsers(false)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
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
			app.storage.SetEmailsKey(id, emailStore)
		}
	}
	app.info.Println("Email list modified")
	respondBool(204, true, gc)
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
	app.debug.Println("Email modification requested")
	users, status, err := app.jf.GetUsers(false)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	ombiEnabled := app.config.Section("ombi").Key("enabled").MustBool(false)
	for _, jfUser := range users {
		id := jfUser.ID
		if address, ok := req[id]; ok {
			var emailStore = EmailAddress{}
			oldEmail, ok := app.storage.GetEmailsKey(id)
			if ok {
				emailStore = oldEmail
			}
			// Auto enable contact by email for newly added addresses
			if !ok || oldEmail.Addr == "" {
				emailStore.Contact = true
			}

			emailStore.Addr = address
			app.storage.SetEmailsKey(id, emailStore)

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

			if ombiEnabled {
				ombiUser, code, err := app.getOmbiUser(id)
				if code == 200 && err == nil {
					ombiUser["emailAddress"] = address
					code, err = app.ombi.ModifyUser(ombiUser)
					if code != 200 || err != nil {
						app.err.Printf("%s: Failed to change ombi email address (%d): %v", ombiUser["userName"].(string), code, err)
					}
				}
			}
		}
	}
	app.info.Println("Email list modified")
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
	applyingFrom := "profile"
	var policy mediabrowser.Policy
	var configuration mediabrowser.Configuration
	var displayprefs map[string]interface{}
	var ombi map[string]interface{}
	if req.From == "profile" {
		// Check profile exists & isn't empty
		profile, ok := app.storage.GetProfileKey(req.Profile)
		if !ok {
			app.err.Printf("Couldn't find profile \"%s\" or profile was empty", req.Profile)
			respond(500, "Couldn't find profile", gc)
			return
		}
		if req.Homescreen {
			if !profile.Homescreen {
				app.err.Printf("No homescreen saved in profile \"%s\"", req.Profile)
				respond(500, "No homescreen template available", gc)
				return
			}
			configuration = profile.Configuration
			displayprefs = profile.Displayprefs
		}
		policy = profile.Policy
		if app.config.Section("ombi").Key("enabled").MustBool(false) {
			if profile.Ombi != nil && len(profile.Ombi) != 0 {
				ombi = profile.Ombi
			}
		}

	} else if req.From == "user" {
		applyingFrom = "user"
		app.jf.CacheExpiry = time.Now()
		user, status, err := app.jf.UserByID(req.ID, false)
		if !(status == 200 || status == 204) || err != nil {
			app.err.Printf("Failed to get user from Jellyfin (%d): %v", status, err)
			respond(500, "Couldn't get user", gc)
			return
		}
		applyingFrom = "\"" + user.Name + "\""
		policy = user.Policy
		if req.Homescreen {
			displayprefs, status, err = app.jf.GetDisplayPreferences(req.ID)
			if !(status == 200 || status == 204) || err != nil {
				app.err.Printf("Failed to get DisplayPrefs (%d): %v", status, err)
				respond(500, "Couldn't get displayprefs", gc)
				return
			}
			configuration = user.Configuration
		}
	}
	app.info.Printf("Applying settings to %d user(s) from %s", len(req.ApplyTo), applyingFrom)
	errors := errorListDTO{
		"policy":     map[string]string{},
		"homescreen": map[string]string{},
		"ombi":       map[string]string{},
	}
	/* Jellyfin doesn't seem to like too many of these requests sent in succession
	and can crash and mess up its database. Issue #160 says this occurs when more
	than 100 users are modified. A delay totalling 500ms between requests is used
	if so. */
	var shouldDelay bool = len(req.ApplyTo) >= 100
	if shouldDelay {
		app.debug.Println("Adding delay between requests for large batch")
	}
	for _, id := range req.ApplyTo {
		status, err := app.jf.SetPolicy(id, policy)
		if !(status == 200 || status == 204) || err != nil {
			errors["policy"][id] = fmt.Sprintf("%d: %s", status, err)
		}
		if shouldDelay {
			time.Sleep(250 * time.Millisecond)
		}
		if req.Homescreen {
			status, err = app.jf.SetConfiguration(id, configuration)
			errorString := ""
			if !(status == 200 || status == 204) || err != nil {
				errorString += fmt.Sprintf("Configuration %d: %v ", status, err)
			} else {
				status, err = app.jf.SetDisplayPreferences(id, displayprefs)
				if !(status == 200 || status == 204) || err != nil {
					errorString += fmt.Sprintf("Displayprefs %d: %v ", status, err)
				}
			}
			if errorString != "" {
				errors["homescreen"][id] = errorString
			}
		}
		if ombi != nil {
			errorString := ""
			user, status, err := app.getOmbiUser(id)
			if status != 200 || err != nil {
				errorString += fmt.Sprintf("Ombi GetUser %d: %v ", status, err)
			} else {
				// newUser := ombi
				// newUser["id"] = user["id"]
				// newUser["userName"] = user["userName"]
				// newUser["alias"] = user["alias"]
				// newUser["emailAddress"] = user["emailAddress"]
				status, err = app.applyOmbiProfile(user, ombi)
				if status != 200 || err != nil {
					errorString += fmt.Sprintf("Apply %d: %v ", status, err)
				}
			}
			if errorString != "" {
				errors["ombi"][id] = errorString
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
