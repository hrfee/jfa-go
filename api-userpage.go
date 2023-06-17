package main

import "github.com/gin-gonic/gin"

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
		if email, ok := app.storage.emails[user.ID]; ok {
			resp.Email.Value = email.Addr
			resp.Email.Enabled = email.Contact
		}
	}

	if discordEnabled {
		resp.Discord = &MyDetailsContactMethodsDTO{}
		if discord, ok := app.storage.discord[user.ID]; ok {
			resp.Discord.Value = RenderDiscordUsername(discord)
			resp.Discord.Enabled = discord.Contact
		}
	}

	if telegramEnabled {
		resp.Telegram = &MyDetailsContactMethodsDTO{}
		if telegram, ok := app.storage.telegram[user.ID]; ok {
			resp.Telegram.Value = telegram.Username
			resp.Telegram.Enabled = telegram.Contact
		}
	}

	if matrixEnabled {
		resp.Matrix = &MyDetailsContactMethodsDTO{}
		if matrix, ok := app.storage.matrix[user.ID]; ok {
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
	cookie, err := gc.Cookie("refresh")
	if err != nil {
		app.debug.Printf("Couldn't get cookies: %s", err)
		respond(500, "Couldn't fetch cookies", gc)
		return
	}
	app.invalidTokens = append(app.invalidTokens, cookie)
	gc.SetCookie("refresh", "invalid", -1, "/my", gc.Request.URL.Hostname(), true, true)
	respondBool(200, true, gc)
}
