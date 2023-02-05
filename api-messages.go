package main

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/ini.v1"
)

// @Summary Get a list of email names and IDs.
// @Produce json
// @Param lang query string false "Language for email titles."
// @Success 200 {object} emailListDTO
// @Router /config/emails [get]
// @Security Bearer
// @tags Configuration
func (app *appContext) GetCustomEmails(gc *gin.Context) {
	lang := gc.Query("lang")
	if _, ok := app.storage.lang.Email[lang]; !ok {
		lang = app.storage.lang.chosenEmailLang
	}
	gc.JSON(200, emailListDTO{
		"UserCreated":       {Name: app.storage.lang.Email[lang].UserCreated["name"], Enabled: app.storage.customEmails.UserCreated.Enabled},
		"InviteExpiry":      {Name: app.storage.lang.Email[lang].InviteExpiry["name"], Enabled: app.storage.customEmails.InviteExpiry.Enabled},
		"PasswordReset":     {Name: app.storage.lang.Email[lang].PasswordReset["name"], Enabled: app.storage.customEmails.PasswordReset.Enabled},
		"UserDeleted":       {Name: app.storage.lang.Email[lang].UserDeleted["name"], Enabled: app.storage.customEmails.UserDeleted.Enabled},
		"UserDisabled":      {Name: app.storage.lang.Email[lang].UserDisabled["name"], Enabled: app.storage.customEmails.UserDisabled.Enabled},
		"UserEnabled":       {Name: app.storage.lang.Email[lang].UserEnabled["name"], Enabled: app.storage.customEmails.UserEnabled.Enabled},
		"InviteEmail":       {Name: app.storage.lang.Email[lang].InviteEmail["name"], Enabled: app.storage.customEmails.InviteEmail.Enabled},
		"WelcomeEmail":      {Name: app.storage.lang.Email[lang].WelcomeEmail["name"], Enabled: app.storage.customEmails.WelcomeEmail.Enabled},
		"EmailConfirmation": {Name: app.storage.lang.Email[lang].EmailConfirmation["name"], Enabled: app.storage.customEmails.EmailConfirmation.Enabled},
		"UserExpired":       {Name: app.storage.lang.Email[lang].UserExpired["name"], Enabled: app.storage.customEmails.UserExpired.Enabled},
	})
}

func (app *appContext) getCustomEmail(id string) *customEmail {
	switch id {
	case "Announcement":
		return &customEmail{}
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
	}
	return nil
}

// @Summary Sets the corresponding custom email.
// @Produce json
// @Param customEmail body customEmail true "Content = email (in markdown)."
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param id path string true "ID of email"
// @Router /config/emails/{id} [post]
// @Security Bearer
// @tags Configuration
func (app *appContext) SetCustomEmail(gc *gin.Context) {
	var req customEmail
	gc.BindJSON(&req)
	id := gc.Param("id")
	if req.Content == "" {
		respondBool(400, false, gc)
		return
	}
	email := app.getCustomEmail(id)
	if email == nil {
		respondBool(400, false, gc)
		return
	}
	email.Content = req.Content
	email.Enabled = true
	if app.storage.storeCustomEmails() != nil {
		respondBool(500, false, gc)
		return
	}
	respondBool(200, true, gc)
}

// @Summary Enable/Disable custom email.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param enable/disable path string true "enable/disable"
// @Param id path string true "ID of email"
// @Router /config/emails/{id}/state/{enable/disable} [post]
// @Security Bearer
// @tags Configuration
func (app *appContext) SetCustomEmailState(gc *gin.Context) {
	id := gc.Param("id")
	s := gc.Param("state")
	enabled := false
	if s == "enable" {
		enabled = true
	} else if s != "disable" {
		respondBool(400, false, gc)
	}
	email := app.getCustomEmail(id)
	if email == nil {
		respondBool(400, false, gc)
		return
	}
	email.Enabled = enabled
	if app.storage.storeCustomEmails() != nil {
		respondBool(500, false, gc)
		return
	}
	respondBool(200, true, gc)
}

// @Summary Returns the custom email (generating it if not set) and list of used variables in it.
// @Produce json
// @Success 200 {object} customEmailDTO
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param id path string true "ID of email"
// @Router /config/emails/{id} [get]
// @Security Bearer
// @tags Configuration
func (app *appContext) GetCustomEmailTemplate(gc *gin.Context) {
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
	email := app.getCustomEmail(id)
	if email == nil {
		app.err.Printf("Failed to get custom email with ID \"%s\"", id)
		respondBool(400, false, gc)
		return
	}
	if id == "WelcomeEmail" {
		conditionals = []string{"{yourAccountWillExpire}"}
		email.Conditionals = conditionals
	}
	content = email.Content
	noContent := content == ""
	if !noContent {
		variables = email.Variables
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
	}
	if err != nil {
		respondBool(500, false, gc)
		return
	}
	if noContent && id != "Announcement" {
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
		email.Variables = variables
	}
	if variables == nil {
		variables = []string{}
	}
	if app.storage.storeCustomEmails() != nil {
		respondBool(500, false, gc)
		return
	}
	mail, err := app.email.constructTemplate("", "<div class=\"preview-content\"></div>", app)
	if err != nil {
		respondBool(500, false, gc)
		return
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
	tokenIndex := -1
	for i, v := range app.telegram.verifiedTokens {
		if v.Token == req.Token {
			tokenIndex = i
			break
		}
	}
	if tokenIndex == -1 {
		respondBool(500, false, gc)
		return
	}
	tgToken := app.telegram.verifiedTokens[tokenIndex]
	tgUser := TelegramUser{
		ChatID:   tgToken.ChatID,
		Username: tgToken.Username,
		Contact:  true,
	}
	if lang, ok := app.telegram.languages[tgToken.ChatID]; ok {
		tgUser.Lang = lang
	}
	if app.storage.telegram == nil {
		app.storage.telegram = map[string]TelegramUser{}
	}
	app.storage.telegram[req.ID] = tgUser
	err := app.storage.storeTelegramUsers()
	if err != nil {
		app.err.Printf("Failed to store Telegram users: %v", err)
	} else {
		app.telegram.verifiedTokens[len(app.telegram.verifiedTokens)-1], app.telegram.verifiedTokens[tokenIndex] = app.telegram.verifiedTokens[tokenIndex], app.telegram.verifiedTokens[len(app.telegram.verifiedTokens)-1]
		app.telegram.verifiedTokens = app.telegram.verifiedTokens[:len(app.telegram.verifiedTokens)-1]
	}
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
	if tgUser, ok := app.storage.telegram[req.ID]; ok {
		change := tgUser.Contact != req.Telegram
		tgUser.Contact = req.Telegram
		app.storage.telegram[req.ID] = tgUser
		if err := app.storage.storeTelegramUsers(); err != nil {
			respondBool(500, false, gc)
			app.err.Printf("Telegram: Failed to store users: %v", err)
			return
		}
		if change {
			msg := ""
			if !req.Telegram {
				msg = " not"
			}
			app.debug.Printf("Telegram: User \"%s\" will%s be notified through Telegram.", tgUser.Username, msg)
		}
	}
	if dcUser, ok := app.storage.discord[req.ID]; ok {
		change := dcUser.Contact != req.Discord
		dcUser.Contact = req.Discord
		app.storage.discord[req.ID] = dcUser
		if err := app.storage.storeDiscordUsers(); err != nil {
			respondBool(500, false, gc)
			app.err.Printf("Discord: Failed to store users: %v", err)
			return
		}
		if change {
			msg := ""
			if !req.Discord {
				msg = " not"
			}
			app.debug.Printf("Discord: User \"%s\" will%s be notified through Discord.", dcUser.Username, msg)
		}
	}
	if mxUser, ok := app.storage.matrix[req.ID]; ok {
		change := mxUser.Contact != req.Matrix
		mxUser.Contact = req.Matrix
		app.storage.matrix[req.ID] = mxUser
		if err := app.storage.storeMatrixUsers(); err != nil {
			respondBool(500, false, gc)
			app.err.Printf("Matrix: Failed to store users: %v", err)
			return
		}
		if change {
			msg := ""
			if !req.Matrix {
				msg = " not"
			}
			app.debug.Printf("Matrix: User \"%s\" will%s be notified through Matrix.", mxUser.UserID, msg)
		}
	}
	if email, ok := app.storage.emails[req.ID]; ok {
		change := email.Contact != req.Email
		email.Contact = req.Email
		app.storage.emails[req.ID] = email
		if err := app.storage.storeEmails(); err != nil {
			respondBool(500, false, gc)
			app.err.Printf("Failed to store emails: %v", err)
			return
		}
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
	tokenIndex := -1
	for i, v := range app.telegram.verifiedTokens {
		if v.Token == pin {
			tokenIndex = i
			break
		}
	}
	// if tokenIndex != -1 {
	// 	length := len(app.telegram.verifiedTokens)
	// 	app.telegram.verifiedTokens[length-1], app.telegram.verifiedTokens[tokenIndex] = app.telegram.verifiedTokens[tokenIndex], app.telegram.verifiedTokens[length-1]
	// 	app.telegram.verifiedTokens = app.telegram.verifiedTokens[:length-1]
	// }
	respondBool(200, tokenIndex != -1, gc)
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
	if _, ok := app.storage.invites[code]; !ok {
		respondBool(401, false, gc)
		return
	}
	pin := gc.Param("pin")
	tokenIndex := -1
	for i, v := range app.telegram.verifiedTokens {
		if v.Token == pin {
			tokenIndex = i
			break
		}
	}
	if app.config.Section("telegram").Key("require_unique").MustBool(false) {
		for _, u := range app.storage.telegram {
			if app.telegram.verifiedTokens[tokenIndex].Username == u.Username {
				respondBool(400, false, gc)
				return
			}
		}
	}
	// if tokenIndex != -1 {
	// 	length := len(app.telegram.verifiedTokens)
	// 	app.telegram.verifiedTokens[length-1], app.telegram.verifiedTokens[tokenIndex] = app.telegram.verifiedTokens[tokenIndex], app.telegram.verifiedTokens[length-1]
	// 	app.telegram.verifiedTokens = app.telegram.verifiedTokens[:length-1]
	// }
	respondBool(200, tokenIndex != -1, gc)
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
	if _, ok := app.storage.invites[code]; !ok {
		respondBool(401, false, gc)
		return
	}
	pin := gc.Param("pin")
	_, ok := app.discord.verifiedTokens[pin]
	if app.config.Section("discord").Key("require_unique").MustBool(false) {
		for _, u := range app.storage.discord {
			if app.discord.verifiedTokens[pin].ID == u.ID {
				delete(app.discord.verifiedTokens, pin)
				respondBool(400, false, gc)
				return
			}
		}
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
	if _, ok := app.storage.invites[code]; !ok {
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
	if _, ok := app.storage.invites[code]; !ok {
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
		for _, u := range app.storage.matrix {
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
	if _, ok := app.storage.invites[code]; !ok {
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
// @tags Other
func (app *appContext) MatrixConnect(gc *gin.Context) {
	var req MatrixConnectUserDTO
	gc.BindJSON(&req)
	if app.storage.matrix == nil {
		app.storage.matrix = map[string]MatrixUser{}
	}
	roomID, encrypted, err := app.matrix.CreateRoom(req.UserID)
	if err != nil {
		app.err.Printf("Matrix: Failed to create room: %v", err)
		respondBool(500, false, gc)
		return
	}
	app.storage.matrix[req.JellyfinID] = MatrixUser{
		UserID:    req.UserID,
		RoomID:    string(roomID),
		Lang:      "en-us",
		Contact:   true,
		Encrypted: encrypted,
	}
	app.matrix.isEncrypted[roomID] = encrypted
	if err := app.storage.storeMatrixUsers(); err != nil {
		app.err.Printf("Failed to store Matrix users: %v", err)
		respondBool(500, false, gc)
		return
	}
	respondBool(200, true, gc)
}

// @Summary Returns a list of matching users from a Discord guild, given a username (discriminator optional).
// @Produce json
// @Success 200 {object} DiscordUsersDTO
// @Failure 400 {object} boolResponse
// @Failure 500 {object} boolResponse
// @Param username path string true "username to search."
// @Router /users/discord/{username} [get]
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
			Name:      u.User.Username + "#" + u.User.Discriminator,
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
	app.storage.discord[req.JellyfinID] = user
	if err := app.storage.storeDiscordUsers(); err != nil {
		app.err.Printf("Failed to store Discord users: %v", err)
		respondBool(500, false, gc)
		return
	}
	linkExistingOmbiDiscordTelegram(app)
	respondBool(200, true, gc)
}

// @Summary unlink a Discord account from a Jellyfin user. Always succeeds.
// @Produce json
// @Success 200 {object} boolResponse
// @Param forUserDTO body forUserDTO true "User's Jellyfin ID."
// @Router /users/discord [delete]
// @Tags Users
func (app *appContext) UnlinkDiscord(gc *gin.Context) {
	var req forUserDTO
	gc.BindJSON(&req)
	/* user, status, err := app.jf.UserByID(req.ID, false)
	if req.ID == "" || status != 200 || err != nil {
		respond(400, "User not found", gc)
		return
	} */
	delete(app.storage.discord, req.ID)
	app.storage.storeDiscordUsers()
	respondBool(200, true, gc)
}

// @Summary unlink a Telegram account from a Jellyfin user. Always succeeds.
// @Produce json
// @Success 200 {object} boolResponse
// @Param forUserDTO body forUserDTO true "User's Jellyfin ID."
// @Router /users/telegram [delete]
// @Tags Users
func (app *appContext) UnlinkTelegram(gc *gin.Context) {
	var req forUserDTO
	gc.BindJSON(&req)
	/* user, status, err := app.jf.UserByID(req.ID, false)
	if req.ID == "" || status != 200 || err != nil {
		respond(400, "User not found", gc)
		return
	} */
	delete(app.storage.telegram, req.ID)
	app.storage.storeTelegramUsers()
	respondBool(200, true, gc)
}

// @Summary unlink a Matrix account from a Jellyfin user. Always succeeds.
// @Produce json
// @Success 200 {object} boolResponse
// @Param forUserDTO body forUserDTO true "User's Jellyfin ID."
// @Router /users/matrix [delete]
// @Tags Users
func (app *appContext) UnlinkMatrix(gc *gin.Context) {
	var req forUserDTO
	gc.BindJSON(&req)
	/* user, status, err := app.jf.UserByID(req.ID, false)
	if req.ID == "" || status != 200 || err != nil {
		respond(400, "User not found", gc)
		return
	} */
	delete(app.storage.matrix, req.ID)
	app.storage.storeMatrixUsers()
	respondBool(200, true, gc)
}
