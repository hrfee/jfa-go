package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// @Summary Returns the logged-in user's Jellyfin ID & Username.
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
	resp.Disabled = user.Policy.IsDisabled

	if exp, ok := app.storage.users[user.ID]; ok {
		resp.Expiry = exp.Unix()
	}

	app.storage.loadEmails()
	app.storage.loadDiscordUsers()
	app.storage.loadMatrixUsers()
	app.storage.loadTelegramUsers()

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

		app.storage.storeEmails()
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
// @tags User Page
func (app *appContext) GetMyPIN(gc *gin.Context) {
	service := gc.Param("service")
	resp := GetMyPINDTO{}
	switch service {
	case "discord":
		resp.PIN = app.discord.NewAuthToken()
		break
	case "telegram":
		resp.PIN = app.telegram.NewAuthToken()
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
// @tags User Page
func (app *appContext) MyDiscordVerifiedInvite(gc *gin.Context) {
	pin := gc.Param("pin")
	dcUser, ok := app.discord.verifiedTokens[pin]
	if !ok {
		respondBool(200, false, gc)
		return
	}
	if app.config.Section("discord").Key("require_unique").MustBool(false) {
		for _, u := range app.storage.GetDiscord() {
			if app.discord.verifiedTokens[pin].ID == u.ID {
				delete(app.discord.verifiedTokens, pin)
				respondBool(400, false, gc)
				return
			}
		}
	}
	existingUser, ok := app.storage.GetDiscordKey(gc.GetString("jfId"))
	if ok {
		dcUser.Lang = existingUser.Lang
		dcUser.Contact = existingUser.Contact
	}
	app.storage.SetDiscordKey(gc.GetString("jfId"), dcUser)
	respondBool(200, true, gc)
}

// @Summary Returns true/false on whether or not your telegram PIN was verified, and assigns the telegram user to you.
// @Produce json
// @Success 200 {object} boolResponse
// @Failure 401 {object} boolResponse
// @Param pin path string true "PIN code to check"
// @Router /my/telegram/verified/{pin} [get]
// @tags User Page
func (app *appContext) MyTelegramVerifiedInvite(gc *gin.Context) {
	pin := gc.Param("pin")
	tokenIndex := -1
	for i, v := range app.telegram.verifiedTokens {
		if v.Token == pin {
			tokenIndex = i
			break
		}
	}
	if tokenIndex == -1 {
		respondBool(200, false, gc)
		return
	}
	if app.config.Section("telegram").Key("require_unique").MustBool(false) {
		for _, u := range app.storage.GetTelegram() {
			if app.telegram.verifiedTokens[tokenIndex].Username == u.Username {
				respondBool(400, false, gc)
				return
			}
		}
	}
	tgUser := TelegramUser{
		ChatID:   app.telegram.verifiedTokens[tokenIndex].ChatID,
		Username: app.telegram.verifiedTokens[tokenIndex].Username,
		Contact:  true,
	}

	existingUser, ok := app.storage.GetTelegramKey(gc.GetString("jfId"))
	if ok {
		tgUser.Lang = existingUser.Lang
		tgUser.Contact = existingUser.Contact
	}
	app.storage.SetTelegramKey(gc.GetString("jfId"), tgUser)
	respondBool(200, true, gc)
}
