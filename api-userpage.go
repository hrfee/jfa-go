package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/lithammer/shortuuid/v3"
	"github.com/timshannon/badgerhold/v4"
)

const (
	REFERRAL_EXPIRY_DAYS = 90
)

// @Summary Returns the logged-in user's Jellyfin ID & Username, and other details.
// @Produce json
// @Success 200 {object} MyDetailsDTO
// @Router /my/details [get]
// @tags User Page
func (app *appContext) MyDetails(gc *gin.Context) {
	resp := MyDetailsDTO{
		Id: gc.GetString("jfId"),
	}

	user, status, err := app.jf.UserByID(resp.Id, false)
	if status != 200 || err != nil {
		app.err.Printf("Failed to get Jellyfin user (%d): %+v\n", status, err)
		respond(500, "Failed to get user", gc)
		return
	}
	resp.Username = user.Name
	resp.Admin = user.Policy.IsAdministrator
	resp.AccountsAdmin = false
	if !app.config.Section("ui").Key("allow_all").MustBool(false) {
		adminOnly := app.config.Section("ui").Key("admin_only").MustBool(true)
		if emailStore, ok := app.storage.GetEmailsKey(resp.Id); ok {
			resp.AccountsAdmin = emailStore.Admin
		}
		resp.AccountsAdmin = resp.AccountsAdmin || (adminOnly && resp.Admin)
	}
	resp.Disabled = user.Policy.IsDisabled

	if exp, ok := app.storage.GetUserExpiryKey(user.ID); ok {
		resp.Expiry = exp.Expiry.Unix()
	}

	if emailEnabled {
		resp.Email = &MyDetailsContactMethodsDTO{}
		if email, ok := app.storage.GetEmailsKey(user.ID); ok && email.Addr != "" {
			resp.Email.Value = email.Addr
			resp.Email.Enabled = email.Contact
		}
	}

	if discordEnabled {
		resp.Discord = &MyDetailsContactMethodsDTO{}
		if discord, ok := app.storage.GetDiscordKey(user.ID); ok {
			resp.Discord.Value = RenderDiscordUsername(discord)
			resp.Discord.Enabled = discord.Contact
		}
	}

	if telegramEnabled {
		resp.Telegram = &MyDetailsContactMethodsDTO{}
		if telegram, ok := app.storage.GetTelegramKey(user.ID); ok {
			resp.Telegram.Value = telegram.Username
			resp.Telegram.Enabled = telegram.Contact
		}
	}

	if matrixEnabled {
		resp.Matrix = &MyDetailsContactMethodsDTO{}
		if matrix, ok := app.storage.GetMatrixKey(user.ID); ok {
			resp.Matrix.Value = matrix.UserID
			resp.Matrix.Enabled = matrix.Contact
		}
	}

	if app.config.Section("user_page").Key("referrals").MustBool(false) {
		// 1. Look for existing template bound to this Jellyfin ID
		//    If one exists, that means its just for us and so we
		//    can use it directly.
		inv := Invite{}
		err := app.storage.db.FindOne(&inv, badgerhold.Where("ReferrerJellyfinID").Eq(resp.Id))
		if err == nil {
			resp.HasReferrals = true
		} else {
			// 2. Look for a template matching the key found in the user storage
			//    Since this key is shared between users in a profile, we make a copy.
			user, ok := app.storage.GetEmailsKey(gc.GetString("jfId"))
			err = app.storage.db.Get(user.ReferralTemplateKey, &inv)
			if ok && err == nil {
				resp.HasReferrals = true
			}
		}
	}

	gc.JSON(200, resp)
}

// @Summary Sets whether to notify yourself through telegram/discord/matrix/email or not.
// @Produce json
// @Param SetContactMethodsDTO body SetContactMethodsDTO true "User's Jellyfin ID and whether or not to notify then through Telegram."
// @Success 200 {object} boolResponse
// @Success 400 {object} boolResponse
// @Success 500 {object} boolResponse
// @Router /my/contact [post]
// @Security Bearer
// @tags User Page
func (app *appContext) SetMyContactMethods(gc *gin.Context) {
	var req SetContactMethodsDTO
	gc.BindJSON(&req)
	req.ID = gc.GetString("jfId")
	if req.ID == "" {
		respondBool(400, false, gc)
		return
	}
	app.setContactMethods(req, gc)
}

// @Summary Logout by deleting refresh token from cookies.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /my/logout [post]
// @Security Bearer
// @tags User Page
func (app *appContext) LogoutUser(gc *gin.Context) {
	cookie, err := gc.Cookie("user-refresh")
	if err != nil {
		app.debug.Printf("Couldn't get cookies: %s", err)
		respond(500, "Couldn't fetch cookies", gc)
		return
	}
	app.invalidTokens = append(app.invalidTokens, cookie)
	gc.SetCookie("refresh", "invalid", -1, "/my", gc.Request.URL.Hostname(), true, true)
	respondBool(200, true, gc)
}

// @Summary confirm an action (e.g. changing an email address.)
// @Produce json
// @Param jwt path string true "jwt confirmation code"
// @Router /my/confirm/{jwt} [post]
// @Success 200 {object} boolResponse
// @Failure 400 {object} stringResponse
// @Failure 404
// @Success 303
// @Failure 500 {object} stringResponse
// @tags User Page
func (app *appContext) ConfirmMyAction(gc *gin.Context) {
	app.confirmMyAction(gc, "")
}

func (app *appContext) confirmMyAction(gc *gin.Context, key string) {
	var claims jwt.MapClaims
	var target ConfirmationTarget
	var id string
	fail := func() {
		gcHTML(gc, 404, "404.html", gin.H{
			"cssClass":       app.cssClass,
			"cssVersion":     cssVersion,
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		})
	}

	// Validate key
	if key == "" {
		key = gc.Param("jwt")
	}
	token, err := jwt.Parse(key, checkToken)
	if err != nil {
		app.err.Printf("Failed to parse key: %s", err)
		fail()
		// respond(500, "unknownError", gc)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		app.err.Printf("Failed to parse key: %s", err)
		fail()
		// respond(500, "unknownError", gc)
		return
	}
	expiry := time.Unix(int64(claims["exp"].(float64)), 0)
	if !(ok && token.Valid && claims["type"].(string) == "confirmation" && expiry.After(time.Now())) {
		app.err.Printf("Invalid key")
		fail()
		// respond(400, "invalidKey", gc)
		return
	}
	target = ConfirmationTarget(int(claims["target"].(float64)))
	id = claims["id"].(string)

	// Perform an Action
	if target == NoOp {
		gc.Redirect(http.StatusSeeOther, "/my/account")
		return
	} else if target == UserEmailChange {
		emailStore, ok := app.storage.GetEmailsKey(id)
		if !ok {
			emailStore = EmailAddress{
				Contact: true,
			}
		}
		emailStore.Addr = claims["email"].(string)
		app.storage.SetEmailsKey(id, emailStore)

		app.storage.SetActivityKey(shortuuid.New(), Activity{
			Type:       ActivityContactLinked,
			UserID:     gc.GetString("jfId"),
			SourceType: ActivityUser,
			Source:     gc.GetString("jfId"),
			Value:      "email",
			Time:       time.Now(),
		}, gc, true)

		if app.config.Section("ombi").Key("enabled").MustBool(false) {
			ombiUser, code, err := app.getOmbiUser(id)
			if code == 200 && err == nil {
				ombiUser["emailAddress"] = claims["email"].(string)
				code, err = app.ombi.ModifyUser(ombiUser)
				if code != 200 || err != nil {
					app.err.Printf("%s: Failed to change ombi email address (%d): %v", ombiUser["userName"].(string), code, err)
				}
			}
		}

		app.info.Println("Email list modified")
		gc.Redirect(http.StatusSeeOther, "/my/account")
		return
	}
}

// @Summary Modify your email address.
// @Produce json
// @Param ModifyMyEmailDTO body ModifyMyEmailDTO true "New email address."
// @Success 200 {object} boolResponse
// @Failure 400 {object} stringResponse
// @Failure 401 {object} stringResponse
// @Failure 500 {object} stringResponse
// @Router /my/email [post]
// @Security Bearer
// @tags User Page
func (app *appContext) ModifyMyEmail(gc *gin.Context) {
	var req ModifyMyEmailDTO
	gc.BindJSON(&req)
	app.debug.Println("Email modification requested")
	if !strings.ContainsRune(req.Email, '@') {
		respond(400, "Invalid Email Address", gc)
		return
	}
	id := gc.GetString("jfId")

	// We'll use the ConfirmMyAction route to do the work, even if we don't need to confirm the address.
	claims := jwt.MapClaims{
		"valid":  true,
		"id":     id,
		"email":  req.Email,
		"type":   "confirmation",
		"target": UserEmailChange,
		"exp":    time.Now().Add(time.Hour).Unix(),
	}
	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	key, err := tk.SignedString([]byte(os.Getenv("JFA_SECRET")))

	if err != nil {
		app.err.Printf("Failed to generate confirmation token: %v", err)
		respond(500, "errorUnknown", gc)
		return
	}

	if emailEnabled && app.config.Section("email_confirmation").Key("enabled").MustBool(false) {
		user, status, err := app.jf.UserByID(id, false)
		name := ""
		if status == 200 && err == nil {
			name = user.Name
		}
		app.debug.Printf("%s: Email confirmation required", id)
		respond(401, "confirmEmail", gc)
		msg, err := app.email.constructConfirmation("", name, key, app, false)
		if err != nil {
			app.err.Printf("%s: Failed to construct confirmation email: %v", name, err)
		} else if err := app.email.send(msg, req.Email); err != nil {
			app.err.Printf("%s: Failed to send user confirmation email: %v", name, err)
		} else {
			app.info.Printf("%s: Sent user confirmation email to \"%s\"", name, req.Email)
		}
		return
	}

	app.confirmMyAction(gc, key)
	return
}

// @Summary Returns a 10-minute, one-use Discord server invite
// @Produce json
// @Success 200 {object} DiscordInviteDTO
// @Failure 400 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param invCode path string true "invite Code"
// @Router /my/discord/invite [get]
// @Security Bearer
// @tags User Page
func (app *appContext) MyDiscordServerInvite(gc *gin.Context) {
	if app.discord.inviteChannelName == "" {
		respondBool(400, false, gc)
		return
	}
	invURL, iconURL := app.discord.NewTempInvite(10*60, 1)
	if invURL == "" {
		respondBool(500, false, gc)
		return
	}
	gc.JSON(200, DiscordInviteDTO{invURL, iconURL})
}

// @Summary Returns a linking PIN for discord/telegram
// @Produce json
// @Success 200 {object} GetMyPINDTO
// @Failure 400 {object} stringResponse
// Param service path string true "discord/telegram"
// @Router /my/pin/{service} [get]
// @Security Bearer
// @tags User Page
func (app *appContext) GetMyPIN(gc *gin.Context) {
	service := gc.Param("service")
	resp := GetMyPINDTO{}
	switch service {
	case "discord":
		resp.PIN = app.discord.NewAssignedAuthToken(gc.GetString("jfId"))
		break
	case "telegram":
		resp.PIN = app.telegram.NewAssignedAuthToken(gc.GetString("jfId"))
		break
	default:
		respond(400, "invalid service", gc)
		return
	}
	gc.JSON(200, resp)
}

// @Summary Returns true/false on whether or not your discord PIN was verified, and assigns the discord user to you.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Param pin path string true "PIN code to check"
// @Router /my/discord/verified/{pin} [get]
// @Security Bearer
// @tags User Page
func (app *appContext) MyDiscordVerifiedInvite(gc *gin.Context) {
	pin := gc.Param("pin")
	dcUser, ok := app.discord.AssignedUserVerified(pin, gc.GetString("jfId"))
	app.discord.DeleteVerifiedUser(pin)
	if !ok {
		respondBool(200, false, gc)
		return
	}
	if app.config.Section("discord").Key("require_unique").MustBool(false) && app.discord.UserExists(dcUser.ID) {
		respondBool(400, false, gc)
		return
	}
	existingUser, ok := app.storage.GetDiscordKey(gc.GetString("jfId"))
	if ok {
		dcUser.Lang = existingUser.Lang
		dcUser.Contact = existingUser.Contact
	}
	app.storage.SetDiscordKey(gc.GetString("jfId"), dcUser)

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactLinked,
		UserID:     gc.GetString("jfId"),
		SourceType: ActivityUser,
		Source:     gc.GetString("jfId"),
		Value:      "discord",
		Time:       time.Now(),
	}, gc, true)

	respondBool(200, true, gc)
}

// @Summary Returns true/false on whether or not your telegram PIN was verified, and assigns the telegram user to you.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Param pin path string true "PIN code to check"
// @Router /my/telegram/verified/{pin} [get]
// @Security Bearer
// @tags User Page
func (app *appContext) MyTelegramVerifiedInvite(gc *gin.Context) {
	pin := gc.Param("pin")
	token, ok := app.telegram.AssignedTokenVerified(pin, gc.GetString("jfId"))
	app.telegram.DeleteVerifiedToken(pin)
	if !ok {
		respondBool(200, false, gc)
		return
	}
	if app.config.Section("telegram").Key("require_unique").MustBool(false) && app.telegram.UserExists(token.Username) {
		respondBool(400, false, gc)
		return
	}
	tgUser := TelegramUser{
		ChatID:   token.ChatID,
		Username: token.Username,
		Contact:  true,
	}
	if lang, ok := app.telegram.languages[tgUser.ChatID]; ok {
		tgUser.Lang = lang
	}

	existingUser, ok := app.storage.GetTelegramKey(gc.GetString("jfId"))
	if ok {
		tgUser.Lang = existingUser.Lang
		tgUser.Contact = existingUser.Contact
	}
	app.storage.SetTelegramKey(gc.GetString("jfId"), tgUser)

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactLinked,
		UserID:     gc.GetString("jfId"),
		SourceType: ActivityUser,
		Source:     gc.GetString("jfId"),
		Value:      "telegram",
		Time:       time.Now(),
	}, gc, true)

	respondBool(200, true, gc)
}

// @Summary Generate and send a new PIN to your given matrix user.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 400 {object} stringResponse
// @Failure 401 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param MatrixSendPINDTO body MatrixSendPINDTO true "User's Matrix ID."
// @Router /my/matrix/user [post]
// @Security Bearer
// @tags User Page
func (app *appContext) MatrixSendMyPIN(gc *gin.Context) {
	var req MatrixSendPINDTO
	gc.BindJSON(&req)
	if req.UserID == "" {
		respond(400, "errorNoUserID", gc)
		return
	}
	if app.config.Section("matrix").Key("require_unique").MustBool(false) {
		for _, u := range app.storage.GetMatrix() {
			if req.UserID == u.UserID {
				respondBool(400, false, gc)
				return
			}
		}
	}

	ok := app.matrix.SendStart(req.UserID)
	if !ok {
		respondBool(500, false, gc)
		return
	}
	respondBool(200, true, gc)
}

// @Summary Check whether your matrix PIN is valid, and link the account to yours if so.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Param pin path string true "PIN code to check"
// @Param invCode path string true "invite Code"
// @Param userID path string true "Matrix User ID"
// @Router /my/matrix/verified/{userID}/{pin} [get]
// @Security Bearer
// @tags User Page
func (app *appContext) MatrixCheckMyPIN(gc *gin.Context) {
	userID := gc.Param("userID")
	pin := gc.Param("pin")
	user, ok := app.matrix.tokens[pin]
	if !ok {
		app.debug.Println("Matrix: PIN not found")
		respondBool(200, false, gc)
		return
	}
	if user.User.UserID != userID {
		app.debug.Println("Matrix: User ID of PIN didn't match")
		respondBool(200, false, gc)
		return
	}

	mxUser := *user.User
	mxUser.Contact = true
	existingUser, ok := app.storage.GetMatrixKey(gc.GetString("jfId"))
	if ok {
		mxUser.Lang = existingUser.Lang
		mxUser.Contact = existingUser.Contact
	}

	app.storage.SetMatrixKey(gc.GetString("jfId"), mxUser)

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactLinked,
		UserID:     gc.GetString("jfId"),
		SourceType: ActivityUser,
		Source:     gc.GetString("jfId"),
		Value:      "matrix",
		Time:       time.Now(),
	}, gc, true)

	delete(app.matrix.tokens, pin)
	respondBool(200, true, gc)
}

// @Summary unlink the Discord account from your Jellyfin user. Always succeeds.
// @Produce json
// @Success 200 {object} boolResponse
// @Router /my/discord [delete]
// @Security Bearer
// @Tags User Page
func (app *appContext) UnlinkMyDiscord(gc *gin.Context) {
	app.storage.DeleteDiscordKey(gc.GetString("jfId"))

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactUnlinked,
		UserID:     gc.GetString("jfId"),
		SourceType: ActivityUser,
		Source:     gc.GetString("jfId"),
		Value:      "discord",
		Time:       time.Now(),
	}, gc, true)

	respondBool(200, true, gc)
}

// @Summary unlink the Telegram account from your Jellyfin user. Always succeeds.
// @Produce json
// @Success 200 {object} boolResponse
// @Router /my/telegram [delete]
// @Security Bearer
// @Tags User Page
func (app *appContext) UnlinkMyTelegram(gc *gin.Context) {
	app.storage.DeleteTelegramKey(gc.GetString("jfId"))

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactUnlinked,
		UserID:     gc.GetString("jfId"),
		SourceType: ActivityUser,
		Source:     gc.GetString("jfId"),
		Value:      "telegram",
		Time:       time.Now(),
	}, gc, true)

	respondBool(200, true, gc)
}

// @Summary unlink the Matrix account from your Jellyfin user. Always succeeds.
// @Produce json
// @Success 200 {object} boolResponse
// @Router /my/matrix [delete]
// @Security Bearer
// @Tags User Page
func (app *appContext) UnlinkMyMatrix(gc *gin.Context) {
	app.storage.DeleteMatrixKey(gc.GetString("jfId"))

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactUnlinked,
		UserID:     gc.GetString("jfId"),
		SourceType: ActivityUser,
		Source:     gc.GetString("jfId"),
		Value:      "matrix",
		Time:       time.Now(),
	}, gc, true)

	respondBool(200, true, gc)
}

// @Summary Generate & send a password reset link if the given username/email/contact method exists. Doesn't give you any info about it's success.
// @Produce json
// @Param address path string true "address/contact method associated w/ your account."
// @Success 204 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /my/password/reset/{address} [post]
// @Tags User Page
func (app *appContext) ResetMyPassword(gc *gin.Context) {
	// All requests should take 1 second, to make it harder to tell if a success occured or not.
	timerWait := make(chan bool)
	cancel := time.AfterFunc(1*time.Second, func() {
		timerWait <- true
	})
	usernameAllowed := app.config.Section("user_page").Key("allow_pwr_username").MustBool(true)
	emailAllowed := app.config.Section("user_page").Key("allow_pwr_email").MustBool(true)
	contactMethodAllowed := app.config.Section("user_page").Key("allow_pwr_contact_method").MustBool(true)
	address := gc.Param("address")
	if address == "" {
		app.debug.Println("Ignoring empty request for PWR")
		cancel.Stop()
		respondBool(400, false, gc)
		return
	}
	var pwr InternalPWR
	var err error

	jfUser, ok := app.ReverseUserSearch(address, usernameAllowed, emailAllowed, contactMethodAllowed)
	if !ok {
		app.debug.Printf("Ignoring PWR request: User not found")

		for range timerWait {
			respondBool(204, true, gc)
			return
		}
		return
	}
	pwr, err = app.GenInternalReset(jfUser.ID)
	if err != nil {
		app.err.Printf("Failed to get user from Jellyfin: %v", err)
		for range timerWait {
			respondBool(204, true, gc)
			return
		}
		return
	}
	if app.internalPWRs == nil {
		app.internalPWRs = map[string]InternalPWR{}
	}
	app.internalPWRs[pwr.PIN] = pwr
	// FIXME: Send to all contact methods
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
		for range timerWait {
			respondBool(204, true, gc)
			return
		}
		return
	} else if err := app.sendByID(msg, jfUser.ID); err != nil {
		app.err.Printf("Failed to send password reset message to \"%s\": %v", address, err)
	} else {
		app.info.Printf("Sent password reset message to \"%s\"", address)
	}
	for range timerWait {
		respondBool(204, true, gc)
		return
	}
}

// @Summary Change your password, given the old one and the new one.
// @Produce json
// @Param ChangeMyPasswordDTO body ChangeMyPasswordDTO true "User's old & new passwords."
// @Success 204 {object} boolResponse
// @Failure 400 {object} PasswordValidation
// @Failure 401 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /my/password [post]
// @Security Bearer
// @Tags User Page
func (app *appContext) ChangeMyPassword(gc *gin.Context) {
	var req ChangeMyPasswordDTO
	gc.BindJSON(&req)
	if req.Old == "" || req.New == "" {
		respondBool(400, false, gc)
	}
	validation := app.validator.validate(req.New)
	for _, val := range validation {
		if !val {
			app.debug.Printf("%s: Change password failed: Invalid password", gc.GetString("jfId"))
			gc.JSON(400, validation)
			return
		}
	}
	user, status, err := app.jf.UserByID(gc.GetString("jfId"), false)
	if status != 200 || err != nil {
		app.err.Printf("Failed to change password: couldn't find user (%d): %+v", status, err)
		respondBool(500, false, gc)
		return
	}
	// Authenticate as user to confirm old password.
	user, status, err = app.authJf.Authenticate(user.Name, req.Old)
	if status != 200 || err != nil {
		respondBool(401, false, gc)
		return
	}
	status, err = app.jf.SetPassword(gc.GetString("jfId"), req.Old, req.New)
	if (status != 200 && status != 204) || err != nil {
		respondBool(500, false, gc)
		return
	}

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityChangePassword,
		UserID:     user.ID,
		SourceType: ActivityUser,
		Source:     user.ID,
		Time:       time.Now(),
	}, gc, true)

	if app.config.Section("ombi").Key("enabled").MustBool(false) {
		func() {
			ombiUser, status, err := app.getOmbiUser(gc.GetString("jfId"))
			if status != 200 || err != nil {
				app.err.Printf("Failed to get user \"%s\" from ombi (%d): %v", user.Name, status, err)
				return
			}
			ombiUser["password"] = req.New
			status, err = app.ombi.ModifyUser(ombiUser)
			if status != 200 || err != nil {
				app.err.Printf("Failed to set password for ombi user \"%s\" (%d): %v", ombiUser["userName"], status, err)
				return
			}
			app.debug.Printf("Reset password for ombi user \"%s\"", ombiUser["userName"])
		}()
	}
	cookie, err := gc.Cookie("user-refresh")
	if err == nil {
		app.invalidTokens = append(app.invalidTokens, cookie)
		gc.SetCookie("refresh", "invalid", -1, "/my", gc.Request.URL.Hostname(), true, true)
	} else {
		app.debug.Printf("Couldn't get cookies: %s", err)
	}
	respondBool(204, true, gc)
}

// @Summary Get or generate a new referral code.
// @Produce json
// @Success 200 {object} GetMyReferralRespDTO
// @Failure 400 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Router /my/referral [get]
// @Security Bearer
// @Tags User Page
func (app *appContext) GetMyReferral(gc *gin.Context) {
	// 1. Look for existing template bound to this Jellyfin ID
	//    If one exists, that means its just for us and so we
	//    can use it directly.
	inv := Invite{}
	err := app.storage.db.FindOne(&inv, badgerhold.Where("ReferrerJellyfinID").Eq(gc.GetString("jfId")))
	if err != nil {
		// 2. Look for a template matching the key found in the user storage
		//    Since this key is shared between users in a profile, we make a copy.
		user, ok := app.storage.GetEmailsKey(gc.GetString("jfId"))
		err = app.storage.db.Get(user.ReferralTemplateKey, &inv)
		if !ok || err != nil || user.ReferralTemplateKey == "" {
			app.debug.Printf("Ignoring referral request, couldn't find template.")
			respondBool(400, false, gc)
			return
		}
		inv.Code = GenerateInviteCode()
		expiryDelta := inv.ValidTill.Sub(inv.Created)
		inv.Created = time.Now()
		if inv.UseReferralExpiry {
			inv.ValidTill = inv.Created.Add(expiryDelta)
		} else {
			inv.ValidTill = inv.Created.Add(REFERRAL_EXPIRY_DAYS * 24 * time.Hour)
		}
		inv.IsReferral = true
		inv.ReferrerJellyfinID = gc.GetString("jfId")
		app.storage.SetInvitesKey(inv.Code, inv)
	} else if time.Now().After(inv.ValidTill) {
		// 3. We found an invite for us, but it's expired.
		//    We delete it from storage, and put it back with a fresh code and expiry.
		//    If UseReferralExpiry is enabled, we delete it and return nothing.
		app.storage.DeleteInvitesKey(inv.Code)
		if inv.UseReferralExpiry {
			user, ok := app.storage.GetEmailsKey(gc.GetString("jfId"))
			if ok {
				user.ReferralTemplateKey = ""
				app.storage.SetEmailsKey(gc.GetString("jfId"), user)
			}
			app.debug.Printf("Ignoring referral request, expired.")
			respondBool(400, false, gc)
			return
		}
		inv.Code = GenerateInviteCode()
		inv.Created = time.Now()
		inv.ValidTill = inv.Created.Add(REFERRAL_EXPIRY_DAYS * 24 * time.Hour)
		app.storage.SetInvitesKey(inv.Code, inv)
	}
	gc.JSON(200, GetMyReferralRespDTO{
		Code:          inv.Code,
		RemainingUses: inv.RemainingUses,
		NoLimit:       inv.NoLimit,
		Expiry:        inv.ValidTill.Unix(),
		UseExpiry:     inv.UseReferralExpiry,
	})
}
