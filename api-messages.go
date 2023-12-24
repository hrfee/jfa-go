package main

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
	adminLang := lang
	if _, ok := app.storage.lang.Admin[lang]; !ok {
		adminLang = app.storage.lang.chosenAdminLang
	}
	list := emailListDTO{
		"UserCreated":       {Name: app.storage.lang.Email[lang].UserCreated["name"], Enabled: app.storage.MustGetCustomContentKey("UserCreated").Enabled},
		"InviteExpiry":      {Name: app.storage.lang.Email[lang].InviteExpiry["name"], Enabled: app.storage.MustGetCustomContentKey("InviteExpiry").Enabled},
		"PasswordReset":     {Name: app.storage.lang.Email[lang].PasswordReset["name"], Enabled: app.storage.MustGetCustomContentKey("PasswordReset").Enabled},
		"UserDeleted":       {Name: app.storage.lang.Email[lang].UserDeleted["name"], Enabled: app.storage.MustGetCustomContentKey("UserDeleted").Enabled},
		"UserDisabled":      {Name: app.storage.lang.Email[lang].UserDisabled["name"], Enabled: app.storage.MustGetCustomContentKey("UserDisabled").Enabled},
		"UserEnabled":       {Name: app.storage.lang.Email[lang].UserEnabled["name"], Enabled: app.storage.MustGetCustomContentKey("UserEnabled").Enabled},
		"InviteEmail":       {Name: app.storage.lang.Email[lang].InviteEmail["name"], Enabled: app.storage.MustGetCustomContentKey("InviteEmail").Enabled},
		"WelcomeEmail":      {Name: app.storage.lang.Email[lang].WelcomeEmail["name"], Enabled: app.storage.MustGetCustomContentKey("WelcomeEmail").Enabled},
		"EmailConfirmation": {Name: app.storage.lang.Email[lang].EmailConfirmation["name"], Enabled: app.storage.MustGetCustomContentKey("EmailConfirmation").Enabled},
		"UserExpired":       {Name: app.storage.lang.Email[lang].UserExpired["name"], Enabled: app.storage.MustGetCustomContentKey("UserExpired").Enabled},
		"UserLogin":         {Name: app.storage.lang.Admin[adminLang].Strings["userPageLogin"], Enabled: app.storage.MustGetCustomContentKey("UserLogin").Enabled},
		"UserPage":          {Name: app.storage.lang.Admin[adminLang].Strings["userPagePage"], Enabled: app.storage.MustGetCustomContentKey("UserPage").Enabled},
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

// No longer needed, these are stored by string keys in the database now.
/* func (app *appContext) getCustomMessage(id string) *CustomContent {
	switch id {
	case "Announcement":
		return &CustomContent{}
	case "UserCreated":
		return &app.storage.customEmails.UserCreated
	case "InviteExpiry":
		return &app.storage.customEmails.InviteExpiry
	case "PasswordReset":
		return &app.storage.customEmails.PasswordReset
	case "UserDeleted":
		return &app.storage.customEmails.UserDeleted
	case "UserDisabled":
		return &app.storage.customEmails.UserDisabled
	case "UserEnabled":
		return &app.storage.customEmails.UserEnabled
	case "InviteEmail":
		return &app.storage.customEmails.InviteEmail
	case "WelcomeEmail":
		return &app.storage.customEmails.WelcomeEmail
	case "EmailConfirmation":
		return &app.storage.customEmails.EmailConfirmation
	case "UserExpired":
		return &app.storage.customEmails.UserExpired
	case "UserLogin":
		return &app.storage.userPage.Login
	case "UserPage":
		return &app.storage.userPage.Page
	}
	return nil
} */

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
	message, ok := app.storage.GetCustomContentKey(id)
	if !ok {
		respondBool(400, false, gc)
		return
	}
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
	lang := app.storage.lang.chosenEmailLang
	id := gc.Param("id")
	var content string
	var err error
	var msg *Message
	var variables []string
	var conditionals []string
	var values map[string]interface{}
	username := app.storage.lang.Email[lang].Strings.get("username")
	emailAddress := app.storage.lang.Email[lang].Strings.get("emailAddress")
	customMessage, ok := app.storage.GetCustomContentKey(id)
	if !ok && id != "Announcement" {
		app.err.Printf("Failed to get custom message with ID \"%s\"", id)
		respondBool(400, false, gc)
		return
	}
	if id == "WelcomeEmail" {
		conditionals = []string{"{yourAccountWillExpire}"}
		customMessage.Conditionals = conditionals
	} else if id == "UserPage" {
		variables = []string{"{username}"}
		customMessage.Variables = variables
	} else if id == "UserLogin" {
		variables = []string{}
		customMessage.Variables = variables
	}
	content = customMessage.Content
	noContent := content == ""
	if !noContent {
		variables = customMessage.Variables
	}
	switch id {
	case "Announcement":
		// Just send the email html
		content = ""
	case "UserCreated":
		if noContent {
			msg, err = app.email.constructCreated("", "", "", Invite{}, app, true)
		}
		values = app.email.createdValues("xxxxxx", username, emailAddress, Invite{}, app, false)
	case "InviteExpiry":
		if noContent {
			msg, err = app.email.constructExpiry("", Invite{}, app, true)
		}
		values = app.email.expiryValues("xxxxxx", Invite{}, app, false)
	case "PasswordReset":
		if noContent {
			msg, err = app.email.constructReset(PasswordReset{}, app, true)
		}
		values = app.email.resetValues(PasswordReset{Pin: "12-34-56", Username: username}, app, false)
	case "UserDeleted":
		if noContent {
			msg, err = app.email.constructDeleted("", app, true)
		}
		values = app.email.deletedValues(app.storage.lang.Email[lang].Strings.get("reason"), app, false)
	case "UserDisabled":
		if noContent {
			msg, err = app.email.constructDisabled("", app, true)
		}
		values = app.email.deletedValues(app.storage.lang.Email[lang].Strings.get("reason"), app, false)
	case "UserEnabled":
		if noContent {
			msg, err = app.email.constructEnabled("", app, true)
		}
		values = app.email.deletedValues(app.storage.lang.Email[lang].Strings.get("reason"), app, false)
	case "InviteEmail":
		if noContent {
			msg, err = app.email.constructInvite("", Invite{}, app, true)
		}
		values = app.email.inviteValues("xxxxxx", Invite{}, app, false)
	case "WelcomeEmail":
		if noContent {
			msg, err = app.email.constructWelcome("", time.Time{}, app, true)
		}
		values = app.email.welcomeValues(username, time.Now(), app, false, true)
	case "EmailConfirmation":
		if noContent {
			msg, err = app.email.constructConfirmation("", "", "", app, true)
		}
		values = app.email.confirmationValues("xxxxxx", username, "xxxxxx", app, false)
	case "UserExpired":
		if noContent {
			msg, err = app.email.constructUserExpired(app, true)
		}
		values = app.email.userExpiredValues(app, false)
	case "UserLogin", "UserPage":
		values = map[string]interface{}{}
	}
	if err != nil {
		respondBool(500, false, gc)
		return
	}
	if noContent && id != "Announcement" && id != "UserPage" && id != "UserLogin" {
		content = msg.Text
		variables = make([]string, strings.Count(content, "{"))
		i := 0
		found := false
		buf := ""
		for _, c := range content {
			if !found && c != '{' && c != '}' {
				continue
			}
			found = true
			buf += string(c)
			if c == '}' {
				found = false
				variables[i] = buf
				buf = ""
				i++
			}
		}
		customMessage.Variables = variables
	}
	if variables == nil {
		variables = []string{}
	}
	app.storage.SetCustomContentKey(id, customMessage)
	var mail *Message
	if id != "UserLogin" && id != "UserPage" {
		mail, err = app.email.constructTemplate("", "<div class=\"preview-content\"></div>", app)
		if err != nil {
			respondBool(500, false, gc)
			return
		}
	} else {
		mail = &Message{
			HTML:     "<div class=\"card ~neutral dark:~d_neutral @low preview-content\"></div>",
			Markdown: "<div class=\"card ~neutral dark:~d_neutral @low preview-content\"></div>",
		}
	}
	gc.JSON(200, customEmailDTO{Content: content, Variables: variables, Conditionals: conditionals, Values: values, HTML: mail.HTML, Plaintext: mail.Text})
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
		ChatID:   tgToken.ChatID,
		Username: tgToken.Username,
		Contact:  true,
	}
	if lang, ok := app.telegram.languages[tgToken.ChatID]; ok {
		tgUser.Lang = lang
	}
	app.storage.SetTelegramKey(req.ID, tgUser)
	linkExistingOmbiDiscordTelegram(app)
	respondBool(200, true, gc)
}

// @Summary Sets whether to notify a user through telegram/discord/matrix/email or not.
// @Produce json
// @Param SetContactMethodsDTO body SetContactMethodsDTO true "User's Jellyfin ID and whether or not to notify then through Telegram."
// @Success 200 {object} boolResponse
// @Success 400 {object} boolResponse
// @Success 500 {object} boolResponse
// @Router /users/contact [post]
// @Security Bearer
// @tags Other
func (app *appContext) SetContactMethods(gc *gin.Context) {
	var req SetContactMethodsDTO
	gc.BindJSON(&req)
	if req.ID == "" {
		respondBool(400, false, gc)
		return
	}
	app.setContactMethods(req, gc)
}

func (app *appContext) setContactMethods(req SetContactMethodsDTO, gc *gin.Context) {
	if tgUser, ok := app.storage.GetTelegramKey(req.ID); ok {
		change := tgUser.Contact != req.Telegram
		tgUser.Contact = req.Telegram
		app.storage.SetTelegramKey(req.ID, tgUser)
		if change {
			msg := ""
			if !req.Telegram {
				msg = " not"
			}
			app.debug.Printf("Telegram: User \"%s\" will%s be notified through Telegram.", tgUser.Username, msg)
		}
	}
	if dcUser, ok := app.storage.GetDiscordKey(req.ID); ok {
		change := dcUser.Contact != req.Discord
		dcUser.Contact = req.Discord
		app.storage.SetDiscordKey(req.ID, dcUser)
		if change {
			msg := ""
			if !req.Discord {
				msg = " not"
			}
			app.debug.Printf("Discord: User \"%s\" will%s be notified through Discord.", dcUser.Username, msg)
		}
	}
	if mxUser, ok := app.storage.GetMatrixKey(req.ID); ok {
		change := mxUser.Contact != req.Matrix
		mxUser.Contact = req.Matrix
		app.storage.SetMatrixKey(req.ID, mxUser)
		if change {
			msg := ""
			if !req.Matrix {
				msg = " not"
			}
			app.debug.Printf("Matrix: User \"%s\" will%s be notified through Matrix.", mxUser.UserID, msg)
		}
	}
	if email, ok := app.storage.GetEmailsKey(req.ID); ok {
		change := email.Contact != req.Email
		email.Contact = req.Email
		app.storage.SetEmailsKey(req.ID, email)
		if change {
			msg := ""
			if !req.Email {
				msg = " not"
			}
			app.debug.Printf("\"%s\" will%s be notified via Email.", email.Addr, msg)
		}
	}
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

// @Summary Returns true/false on whether or not a telegram PIN was verified. Requires invite code.
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
		app.discord.DeleteVerifiedUser(pin)
		respondBool(400, false, gc)
		return
	}
	respondBool(200, ok, gc)
}

// @Summary Returns true/false on whether or not a discord PIN was verified. Requires invite code.
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
	if ok && app.config.Section("discord").Key("require_unique").MustBool(false) && app.discord.UserExists(user.ID) {
		delete(app.discord.verifiedTokens, pin)
		respondBool(400, false, gc)
		return
	}
	respondBool(200, ok, gc)
}

// @Summary Returns a 10-minute, one-use Discord server invite
// @Produce json
// @Success 200 {object} DiscordInviteDTO
// @Failure 400 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param invCode path string true "invite Code"
// @Router /invite/{invCode}/discord/invite [get]
// @tags Other
func (app *appContext) DiscordServerInvite(gc *gin.Context) {
	if app.discord.inviteChannelName == "" {
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

// @Summary Generate and send a new PIN to a specified Matrix user.
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

// @Summary Check whether a matrix PIN is valid, and mark the token as verified if so. Requires invite code.
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
		app.debug.Println("Matrix: Invite code was invalid")
		respondBool(401, false, gc)
		return
	}
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
		app.err.Printf("Matrix: Failed to generate token: %v", err)
		respond(401, "Unauthorized", gc)
		return
	}
	tempConfig, _ := ini.Load(app.configPath)
	matrix := tempConfig.Section("matrix")
	matrix.Key("enabled").SetValue("true")
	matrix.Key("homeserver").SetValue(req.Homeserver)
	matrix.Key("token").SetValue(token)
	matrix.Key("user_id").SetValue(req.Username)
	if err := tempConfig.SaveTo(app.configPath); err != nil {
		app.err.Printf("Failed to save config to \"%s\": %v", app.configPath, err)
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
	roomID, encrypted, err := app.matrix.CreateRoom(req.UserID)
	if err != nil {
		app.err.Printf("Matrix: Failed to create room: %v", err)
		respondBool(500, false, gc)
		return
	}
	app.storage.SetMatrixKey(req.JellyfinID, MatrixUser{
		UserID:    req.UserID,
		RoomID:    string(roomID),
		Lang:      "en-us",
		Contact:   true,
		Encrypted: encrypted,
	})
	app.matrix.isEncrypted[roomID] = encrypted
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

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactLinked,
		UserID:     req.JellyfinID,
		SourceType: ActivityAdmin,
		Source:     gc.GetString("jfId"),
		Value:      "discord",
		Time:       time.Now(),
	}, gc, false)

	linkExistingOmbiDiscordTelegram(app)
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

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactUnlinked,
		UserID:     req.ID,
		SourceType: ActivityAdmin,
		Source:     gc.GetString("jfId"),
		Value:      "discord",
		Time:       time.Now(),
	}, gc, false)

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

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityContactUnlinked,
		UserID:     req.ID,
		SourceType: ActivityAdmin,
		Source:     gc.GetString("jfId"),
		Value:      "telegram",
		Time:       time.Now(),
	}, gc, false)

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

	respondBool(200, true, gc)
}
