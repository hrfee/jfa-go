package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/common"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/lithammer/shortuuid/v3"
	"gopkg.in/ini.v1"
)

// @Summary Get a list of email names and IDs.
// @Produce json
// @Param lang query string false "Language for email titles."
// @Success 200 {object} emailListDTO
// @Router /config/emails [get]
// @Security Bearer
// @tags Configuration
func (app *appContext) GetCustomContent(gc *gin.Context) {
	lang := gc.Query("lang")
	if _, ok := app.storage.lang.Email[lang]; !ok {
		lang = app.storage.lang.chosenEmailLang
	}
	list := emailListDTO{}
	for _, cc := range customContent {
		if cc.ContentType == CustomTemplate {
			continue
		}
		ccDescription := emailListEl{Name: cc.DisplayName(&app.storage.lang, lang), Enabled: app.storage.MustGetCustomContentKey(cc.Name).Enabled}
		if cc.Description != nil {
			ccDescription.Description = cc.Description(&app.storage.lang, lang)
		}
		list[cc.Name] = ccDescription
	}

	filter := gc.Query("filter")
	if filter == "user" {
		list = emailListDTO{"UserLogin": list["UserLogin"], "UserPage": list["UserPage"]}
	} else {
		delete(list, "UserLogin")
		delete(list, "UserPage")
	}

	gc.JSON(200, list)
}

// @Summary Sets the corresponding custom content.
// @Produce json
// @Param CustomContent body CustomContent true "Content = email (in markdown)."
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param id path string true "ID of content"
// @Router /config/emails/{id} [post]
// @Security Bearer
// @tags Configuration
func (app *appContext) SetCustomMessage(gc *gin.Context) {
	var req CustomContent
	gc.BindJSON(&req)
	id := gc.Param("id")
	if req.Content == "" {
		respondBool(400, false, gc)
		return
	}
	_, ok := customContent[id]
	if !ok {
		respondBool(400, false, gc)
		return
	}
	message, ok := app.storage.GetCustomContentKey(id)
	message.Content = req.Content
	message.Enabled = true
	app.storage.SetCustomContentKey(id, message)
	respondBool(200, true, gc)
}

// @Summary Enable/Disable custom content.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param enable/disable path string true "enable/disable"
// @Param id path string true "ID of email"
// @Router /config/emails/{id}/state/{enable/disable} [post]
// @Security Bearer
// @tags Configuration
func (app *appContext) SetCustomMessageState(gc *gin.Context) {
	id := gc.Param("id")
	s := gc.Param("state")
	enabled := false
	if s == "enable" {
		enabled = true
	} else if s != "disable" {
		respondBool(400, false, gc)
	}
	message, ok := app.storage.GetCustomContentKey(id)
	if !ok {
		respondBool(400, false, gc)
		return
	}
	message.Enabled = enabled
	app.storage.SetCustomContentKey(id, message)
	respondBool(200, true, gc)
}

// @Summary Returns the custom content/message (generating it if not set) and list of used variables in it.
// @Produce json
// @Success 200 {object} customEmailDTO
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param id path string true "ID of email"
// @Router /config/emails/{id} [get]
// @Security Bearer
// @tags Configuration
func (app *appContext) GetCustomMessageTemplate(gc *gin.Context) {
	id := gc.Param("id")
	var err error
	contentInfo, ok := customContent[id]
	// FIXME: Add announcement to customContent
	if !ok && id != "Announcement" {
		app.err.Printf(lm.FailedGetCustomMessage, id)
		respondBool(400, false, gc)
		return
	}

	content, ok := app.storage.GetCustomContentKey(id)

	if contentInfo.Variables == nil {
		contentInfo.Variables = []string{}
	}
	if contentInfo.Conditionals == nil {
		contentInfo.Conditionals = []string{}
	}
	if contentInfo.Placeholders == nil {
		contentInfo.Placeholders = map[string]any{}
	}

	// Generate content from real email, if the user hasn't already customised this message.
	if content.Content == "" {
		var msg *Message
		switch id {
		// FIXME: Add announcement to customContent
		case "UserCreated":
			msg, err = app.email.constructCreated("", "", time.Time{}, Invite{}, true)
		case "InviteExpiry":
			msg, err = app.email.constructExpiry(Invite{}, true)
		case "PasswordReset":
			msg, err = app.email.constructReset(PasswordReset{}, true)
		case "UserDeleted":
			msg, err = app.email.constructDeleted("", "", true)
		case "UserDisabled":
			msg, err = app.email.constructDisabled("", "", true)
		case "UserEnabled":
			msg, err = app.email.constructEnabled("", "", true)
		case "UserExpiryAdjusted":
			msg, err = app.email.constructExpiryAdjusted("", time.Time{}, "", true)
		case "ExpiryReminder":
			msg, err = app.email.constructExpiryReminder("", time.Now().AddDate(0, 0, 3), true)
		case "InviteEmail":
			msg, err = app.email.constructInvite(Invite{Code: ""}, true)
		case "WelcomeEmail":
			msg, err = app.email.constructWelcome("", time.Time{}, true)
		case "EmailConfirmation":
			msg, err = app.email.constructConfirmation("", "", "", true)
		case "UserExpired":
			msg, err = app.email.constructUserExpired("", true)
		case "Announcement":
		case "UserPage":
		case "UserLogin":
		case "PostSignupCard":
		case "PreSignupCard":
			// These don't have any example content
			msg = nil
		}
		if err != nil {
			respondBool(500, false, gc)
			return
		}
		if msg != nil {
			content.Content = msg.Text
		}
	}

	var mail *Message = nil
	if contentInfo.ContentType == CustomMessage {
		mail, err = app.email.construct(EmptyCustomContent, CustomContent{
			Name:    EmptyCustomContent.Name,
			Enabled: true,
			Content: "<div class=\"preview-content\"></div>",
		}, map[string]any{})
		if err != nil {
			respondBool(500, false, gc)
			return
		}
	} else if id == "PostSignupCard" {
		// Specific workaround for the currently-unique "Post signup card".
		// Source content from "Success Message" setting.
		if content.Content == "" {
			content.Content = "# " + app.storage.lang.User[app.storage.lang.chosenUserLang].Strings.get("successHeader") + "\n" + app.config.Section("ui").Key("success_message").String()
			if app.config.Section("user_page").Key("enabled").MustBool(false) {
				content.Content += "\n\n<br>\n" + app.storage.lang.User[app.storage.lang.chosenUserLang].Strings.template("userPageSuccessMessage", tmpl{
					"myAccount": "[" + app.storage.lang.User[app.storage.lang.chosenUserLang].Strings.get("myAccount") + "]({myAccountURL})",
				})
			}
		}
		mail = &Message{
			HTML: "<div class=\"card ~neutral dark:~d_neutral @low\"><div class=\"preview-content\"></div><br><button class=\"button ~urge dark:~d_urge @low full-width center supra submit\">" + app.storage.lang.User[app.storage.lang.chosenUserLang].Strings.get("continue") + "</a></div>",
		}
		mail.Markdown = mail.HTML
	} else if contentInfo.ContentType == CustomCard {
		mail = &Message{
			HTML: "<div class=\"card ~neutral dark:~d_neutral @low preview-content\"></div>",
		}
		mail.Markdown = mail.HTML
	} else {
		app.err.Printf("unknown custom content type %d", contentInfo.ContentType)
	}
	gc.JSON(200, customEmailDTO{Content: content.Content, Variables: contentInfo.Variables, Conditionals: contentInfo.Conditionals, Values: contentInfo.Placeholders, HTML: mail.HTML, Plaintext: mail.Text})
}

// @Summary Returns a new Telegram verification PIN, and the bot username.
// @Produce json
// @Success 200 {object} telegramPinDTO
// @Router /telegram/pin [get]
// @Security Bearer
// @tags Other
func (app *appContext) TelegramGetPin(gc *gin.Context) {
	gc.JSON(200, telegramPinDTO{
		Token:    app.telegram.NewAuthToken(),
		Username: app.telegram.username,
	})
}

// @Summary Link a Jellyfin & Telegram user together via a verification PIN.
// @Produce json
// @Param telegramSetDTO body telegramSetDTO true "Token and user's Jellyfin ID."
// @Success 200 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Router /users/telegram [post]
// @Security Bearer
// @tags Other
func (app *appContext) TelegramAddUser(gc *gin.Context) {
	var req telegramSetDTO
	gc.BindJSON(&req)
	if req.Token == "" || req.ID == "" {
		respondBool(400, false, gc)
		return
	}
	tgToken, ok := app.telegram.TokenVerified(req.Token)
	app.telegram.DeleteVerifiedToken(req.Token)
	if !ok {
		respondBool(500, false, gc)
		return
	}
	tgUser := TelegramUser{
		TelegramVerifiedToken: TelegramVerifiedToken{
			ChatID:   tgToken.ChatID,
			Username: tgToken.Username,
		},
		Contact: true,
	}
	if lang, ok := app.telegram.languages[tgToken.ChatID]; ok {
		tgUser.Lang = lang
	}
	app.storage.SetTelegramKey(req.ID, tgUser)

	for _, tps := range app.thirdPartyServices {
		if err := tps.SetContactMethods(req.ID, nil, nil, &tgUser, &common.ContactPreferences{
			Telegram: &tgUser.Contact,
		}); err != nil {
			app.err.Printf(lm.FailedSyncContactMethods, tps.Name(), err)
		}
	}

	app.InvalidateWebUserCache()
	respondBool(200, true, gc)
}

// @Summary Sets whether to notify a user through telegram/discord/matrix/email or not.
// @Produce json
// @Param SetContactPreferencesDTO body SetContactPreferencesDTO true "User's Jellyfin ID and whether or not to notify then through Telegram."
// @Success 200 {object} boolResponse
// @Success 400 {object} boolResponse
// @Success 500 {object} boolResponse
// @Router /users/contact [post]
// @Security Bearer
// @tags Other
func (app *appContext) SetContactMethods(gc *gin.Context) {
	var req SetContactPreferencesDTO
	gc.BindJSON(&req)
	if req.ID == "" {
		respondBool(400, false, gc)
		return
	}
	app.setContactPreferences(req, gc)
}

func (app *appContext) setContactPreferences(req SetContactPreferencesDTO, gc *gin.Context) {
	contactPrefs := common.ContactPreferences{}
	if tgUser, ok := app.storage.GetTelegramKey(req.ID); ok {
		change := tgUser.Contact != req.Telegram
		tgUser.Contact = req.Telegram
		app.storage.SetTelegramKey(req.ID, tgUser)
		if change {
			app.debug.Printf(lm.SetContactPrefForService, lm.Telegram, tgUser.Username, req.Telegram)
			contactPrefs.Telegram = &req.Telegram
		}
	}
	if dcUser, ok := app.storage.GetDiscordKey(req.ID); ok {
		change := dcUser.Contact != req.Discord
		dcUser.Contact = req.Discord
		app.storage.SetDiscordKey(req.ID, dcUser)
		if change {
			app.debug.Printf(lm.SetContactPrefForService, lm.Discord, dcUser.Username, req.Discord)
			contactPrefs.Discord = &req.Discord
		}
	}
	if mxUser, ok := app.storage.GetMatrixKey(req.ID); ok {
		change := mxUser.Contact != req.Matrix
		mxUser.Contact = req.Matrix
		app.storage.SetMatrixKey(req.ID, mxUser)
		if change {
			app.debug.Printf(lm.SetContactPrefForService, lm.Matrix, mxUser.UserID, req.Matrix)
			contactPrefs.Matrix = &req.Matrix
		}
	}
	if email, ok := app.storage.GetEmailsKey(req.ID); ok {
		change := email.Contact != req.Email
		email.Contact = req.Email
		app.storage.SetEmailsKey(req.ID, email)
		if change {
			app.debug.Printf(lm.SetContactPrefForService, lm.Email, email.Addr, req.Email)
			contactPrefs.Email = &req.Email
		}
	}

	for _, tps := range app.thirdPartyServices {
		if err := tps.SetContactMethods(req.ID, nil, nil, nil, &contactPrefs); err != nil {
			app.err.Printf(lm.FailedSyncContactMethods, tps.Name(), err)
		}
	}
	app.InvalidateWebUserCache()
	respondBool(200, true, gc)
}

// @Summary Returns true/false on whether or not a telegram PIN was verified. Requires bearer auth.
// @Produce json
// @Success 200 {object} boolResponse
// @Param pin path string true "PIN code to check"
// @Router /telegram/verified/{pin} [get]
// @Security Bearer
// @tags Other
func (app *appContext) TelegramVerified(gc *gin.Context) {
	pin := gc.Param("pin")
	_, ok := app.telegram.TokenVerified(pin)
	respondBool(200, ok, gc)
}

// @Summary Returns true/false on whether or not a telegram PIN was verified. Requires invite code. NOTE: "/invite" might have been changed in Settings > URL Paths.
// @Produce json
// @Success 200 {object} boolResponse
// @Success 401 {object} boolResponse
// @Param pin path string true "PIN code to check"
// @Param invCode path string true "invite Code"
// @Router /invite/{invCode}/telegram/verified/{pin} [get]
// @tags Other
func (app *appContext) TelegramVerifiedInvite(gc *gin.Context) {
	code := gc.Param("invCode")
	if _, ok := app.storage.GetInvitesKey(code); !ok {
		respondBool(401, false, gc)
		return
	}
	pin := gc.Param("pin")
	token, ok := app.telegram.TokenVerified(pin)
	if ok && app.config.Section("telegram").Key("require_unique").MustBool(false) && app.telegram.UserExists(token.Username) {
		app.discord.DeleteVerifiedToken(pin)
		respondBool(400, false, gc)
		return
	}
	respondBool(200, ok, gc)
}

// @Summary Returns true/false on whether or not a discord PIN was verified. Requires invite code. NOTE: "/invite" might have been changed in Settings > URL Paths.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Param pin path string true "PIN code to check"
// @Param invCode path string true "invite Code"
// @Router /invite/{invCode}/discord/verified/{pin} [get]
// @tags Other
func (app *appContext) DiscordVerifiedInvite(gc *gin.Context) {
	code := gc.Param("invCode")
	if _, ok := app.storage.GetInvitesKey(code); !ok {
		respondBool(401, false, gc)
		return
	}
	pin := gc.Param("pin")
	user, ok := app.discord.UserVerified(pin)
	if ok && app.config.Section("discord").Key("require_unique").MustBool(false) && app.discord.UserExists(user.MethodID().(string)) {
		delete(app.discord.verifiedTokens, pin)
		respondBool(400, false, gc)
		return
	}
	respondBool(200, ok, gc)
}

// @Summary Returns a 10-minute, one-use Discord server invite. NOTE: "/invite" might have been changed in Settings > URL Paths.
// @Produce json
// @Success 200 {object} DiscordInviteDTO
// @Failure 400 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param invCode path string true "invite Code"
// @Router /invite/{invCode}/discord/invite [get]
// @tags Other
func (app *appContext) DiscordServerInvite(gc *gin.Context) {
	if app.discord.InviteChannel.Name == "" {
		respondBool(400, false, gc)
		return
	}
	code := gc.Param("invCode")
	if _, ok := app.storage.GetInvitesKey(code); !ok {
		respondBool(401, false, gc)
		return
	}
	invURL, iconURL := app.discord.NewTempInvite(10*60, 1)
	if invURL == "" {
		respondBool(500, false, gc)
		return
	}
	gc.JSON(200, DiscordInviteDTO{invURL, iconURL})
}

// @Summary Generate and send a new PIN to a specified Matrix user. NOTE: "/invite" might have been changed in Settings > URL Paths.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 400 {object} stringResponse
// @Failure 401 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param invCode path string true "invite Code"
// @Param MatrixSendPINDTO body MatrixSendPINDTO true "User's Matrix ID."
// @Router /invite/{invCode}/matrix/user [post]
// @tags Other
func (app *appContext) MatrixSendPIN(gc *gin.Context) {
	code := gc.Param("invCode")
	if _, ok := app.storage.GetInvitesKey(code); !ok {
		respondBool(401, false, gc)
		return
	}
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

// @Summary Check whether a matrix PIN is valid, and mark the token as verified if so. Requires invite code. NOTE: "/invite" might have been changed in Settings > URL Paths.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Param pin path string true "PIN code to check"
// @Param invCode path string true "invite Code"
// @Param userID path string true "Matrix User ID"
// @Router /invite/{invCode}/matrix/verified/{userID}/{pin} [get]
// @tags Other
func (app *appContext) MatrixCheckPIN(gc *gin.Context) {
	code := gc.Param("invCode")
	if _, ok := app.storage.GetInvitesKey(code); !ok {
		app.debug.Printf(lm.InvalidInviteCode, code)
		respondBool(401, false, gc)
		return
	}
	userID := gc.Param("userID")
	pin := gc.Param("pin")
	user, ok := app.matrix.tokens[pin]
	if !ok {
		app.debug.Printf(lm.InvalidPIN, pin)
		respondBool(200, false, gc)
		return
	}
	if user.User.UserID != userID {
		app.debug.Printf(lm.UnauthorizedPIN, pin)
		respondBool(200, false, gc)
		return
	}
	user.Verified = true
	app.matrix.tokens[pin] = user
	respondBool(200, true, gc)
}

// @Summary Generates a Matrix access token from a username and password.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 400 {object} stringResponse
// @Failure 401 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param MatrixLoginDTO body MatrixLoginDTO true "Username & password."
// @Router /matrix/login [post]
// @Security Bearer
// @tags Other
func (app *appContext) MatrixLogin(gc *gin.Context) {
	var req MatrixLoginDTO
	gc.BindJSON(&req)
	if req.Username == "" || req.Password == "" {
		respond(400, "errorLoginBlank", gc)
		return
	}
	token, err := app.matrix.generateAccessToken(req.Homeserver, req.Username, req.Password)
	if err != nil {
		app.err.Printf(lm.FailedGenerateToken, err)
		respond(401, "Unauthorized", gc)
		return
	}
	tempConfig, _ := ini.ShadowLoad(app.configPath)
	matrix := tempConfig.Section("matrix")
	matrix.Key("enabled").SetValue("true")
	matrix.Key("homeserver").SetValue(req.Homeserver)
	matrix.Key("token").SetValue(token)
	matrix.Key("user_id").SetValue(req.Username)
	if err := tempConfig.SaveTo(app.configPath); err != nil {
		app.err.Printf(lm.FailedWriting, app.configPath, err)
		respondBool(500, false, gc)
		return
	}
	respondBool(200, true, gc)
}

// @Summary Links a Matrix user to a Jellyfin account via user IDs. Notifications are turned on by default.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param MatrixConnectUserDTO body MatrixConnectUserDTO true "User's Jellyfin ID & Matrix user ID."
// @Router /users/matrix [post]
// @Security Bearer
// @tags Other
func (app *appContext) MatrixConnect(gc *gin.Context) {
	var req MatrixConnectUserDTO
	gc.BindJSON(&req)
	if app.storage.GetMatrix() == nil {
		app.storage.deprecatedMatrix = matrixStore{}
	}
	roomID, err := app.matrix.CreateRoom(req.UserID)
	if err != nil {
		app.err.Printf(lm.FailedCreateRoom, err)
		respondBool(500, false, gc)
		return
	}
	app.storage.SetMatrixKey(req.JellyfinID, MatrixUser{
		UserID:  req.UserID,
		RoomID:  string(roomID),
		Lang:    "en-us",
		Contact: true,
	})
	app.InvalidateWebUserCache()
	respondBool(200, true, gc)
}

// @Summary Returns a list of matching users from a Discord guild, given a username (discriminator optional).
// @Produce json
// @Success 200 {object} DiscordUsersDTO
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param username path string true "username to search."
// @Router /users/discord/{username} [get]
// @Security Bearer
// @tags Other
func (app *appContext) DiscordGetUsers(gc *gin.Context) {
	name := gc.Param("username")
	if name == "" {
		respondBool(400, false, gc)
		return
	}
	users := app.discord.GetUsers(name)
	resp := DiscordUsersDTO{Users: make([]DiscordUserDTO, len(users))}
	for i, u := range users {
		resp.Users[i] = DiscordUserDTO{
			Name:      RenderDiscordUsername(u.User),
			ID:        u.User.ID,
			AvatarURL: u.User.AvatarURL("32"),
		}
	}
	gc.JSON(200, resp)
}

// @Summary Links a Discord account to a Jellyfin account via user IDs. Notifications are turned on by default.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param DiscordConnectUserDTO body DiscordConnectUserDTO true "User's Jellyfin ID & Discord ID."
// @Router /users/discord [post]
// @Security Bearer
// @tags Other
func (app *appContext) DiscordConnect(gc *gin.Context) {
	var req DiscordConnectUserDTO
	gc.BindJSON(&req)
	if req.JellyfinID == "" || req.DiscordID == "" {
		respondBool(400, false, gc)
		return
	}
	user, ok := app.discord.NewUser(req.DiscordID)
	if !ok {
		respondBool(500, false, gc)
		return
	}

	app.storage.SetDiscordKey(req.JellyfinID, user)

	for _, tps := range app.thirdPartyServices {
		if err := tps.SetContactMethods(req.JellyfinID, nil, &user, nil, &common.ContactPreferences{
			Discord: &user.Contact,
		}); err != nil {
			app.err.Printf(lm.FailedSyncContactMethods, tps.Name(), err)
		}
	}

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactLinked,
		UserID:     req.JellyfinID,
		SourceType: ActivityAdmin,
		Source:     gc.GetString("jfId"),
		Value:      "discord",
		Time:       time.Now(),
	}, gc, false)

	linkExistingOmbiDiscordTelegram(app)
	app.InvalidateWebUserCache()
	respondBool(200, true, gc)
}

// @Summary unlink a Discord account from a Jellyfin user. Always succeeds.
// @Produce json
// @Success 200 {object} boolResponse
// @Param forUserDTO body forUserDTO true "User's Jellyfin ID."
// @Router /users/discord [delete]
// @Security Bearer
// @Tags Users
func (app *appContext) UnlinkDiscord(gc *gin.Context) {
	var req forUserDTO
	gc.BindJSON(&req)
	/* user, status, err := app.jf.UserByID(req.ID, false)
	if req.ID == "" || status != 200 || err != nil {
		respond(400, "User not found", gc)
		return
	} */
	app.storage.DeleteDiscordKey(req.ID)

	contact := false

	for _, tps := range app.thirdPartyServices {
		if err := tps.SetContactMethods(req.ID, nil, EmptyDiscordUser(), nil, &common.ContactPreferences{
			Discord: &contact,
		}); err != nil {
			app.err.Printf(lm.FailedSyncContactMethods, tps.Name(), err)
		}
	}

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactUnlinked,
		UserID:     req.ID,
		SourceType: ActivityAdmin,
		Source:     gc.GetString("jfId"),
		Value:      "discord",
		Time:       time.Now(),
	}, gc, false)

	app.InvalidateWebUserCache()
	respondBool(200, true, gc)
}

// @Summary unlink a Telegram account from a Jellyfin user. Always succeeds.
// @Produce json
// @Success 200 {object} boolResponse
// @Param forUserDTO body forUserDTO true "User's Jellyfin ID."
// @Router /users/telegram [delete]
// @Security Bearer
// @Tags Users
func (app *appContext) UnlinkTelegram(gc *gin.Context) {
	var req forUserDTO
	gc.BindJSON(&req)
	/* user, status, err := app.jf.UserByID(req.ID, false)
	if req.ID == "" || status != 200 || err != nil {
		respond(400, "User not found", gc)
		return
	} */
	app.storage.DeleteTelegramKey(req.ID)

	contact := false

	for _, tps := range app.thirdPartyServices {
		if err := tps.SetContactMethods(req.ID, nil, nil, EmptyTelegramUser(), &common.ContactPreferences{
			Telegram: &contact,
		}); err != nil {
			app.err.Printf(lm.FailedSyncContactMethods, tps.Name(), err)
		}
	}

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactUnlinked,
		UserID:     req.ID,
		SourceType: ActivityAdmin,
		Source:     gc.GetString("jfId"),
		Value:      "telegram",
		Time:       time.Now(),
	}, gc, false)

	app.InvalidateWebUserCache()
	respondBool(200, true, gc)
}

// @Summary unlink a Matrix account from a Jellyfin user. Always succeeds.
// @Produce json
// @Success 200 {object} boolResponse
// @Param forUserDTO body forUserDTO true "User's Jellyfin ID."
// @Router /users/matrix [delete]
// @Security Bearer
// @Tags Users
func (app *appContext) UnlinkMatrix(gc *gin.Context) {
	var req forUserDTO
	gc.BindJSON(&req)
	/* user, status, err := app.jf.UserByID(req.ID, false)
	if req.ID == "" || status != 200 || err != nil {
		respond(400, "User not found", gc)
		return
	} */
	app.storage.DeleteMatrixKey(req.ID)

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactUnlinked,
		UserID:     req.ID,
		SourceType: ActivityAdmin,
		Source:     gc.GetString("jfId"),
		Value:      "matrix",
		Time:       time.Now(),
	}, gc, false)

	app.InvalidateWebUserCache()
	respondBool(200, true, gc)
}
