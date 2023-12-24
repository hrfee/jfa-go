package main

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (app *appContext) userAuth() gin.HandlerFunc {
	return app.userAuthenticate
}

func (app *appContext) userAuthenticate(gc *gin.Context) {
	jellyfinLogin := app.config.Section("ui").Key("jellyfin_login").MustBool(true)
	if !jellyfinLogin {
		app.err.Println("Enable Jellyfin Login to use the User Page feature.")
		respond(500, "Contact Admin", gc)
		return
	}
	claims, ok := app.decodeValidateAuthHeader(gc)
	if !ok {
		return
	}

	// user id can be nil for all we care, we just want the Jellyfin ID
	jfID := claims["jfid"].(string)

	gc.Set("jfId", jfID)
	gc.Set("userMode", true)
	app.debug.Println("Auth succeeded")
	gc.Next()
}

// @Summary Grabs an user-access token using username & password.
// @description Has limited access to API routes, used to display the user's personal page.
// @Produce json
// @Success 200 {object} getTokenDTO
// @Failure 401 {object} stringResponse
// @Router /my/token/login [get]
// @tags Auth
// @Security getUserTokenAuth
func (app *appContext) getUserTokenLogin(gc *gin.Context) {
	if !app.config.Section("ui").Key("jellyfin_login").MustBool(true) {
		app.err.Println("Enable Jellyfin Login to use the User Page feature.")
		respond(500, "Contact Admin", gc)
		return
	}
	app.logIpInfo(gc, true, "UserToken requested (login attempt)")
	username, password, ok := app.decodeValidateLoginHeader(gc, true)
	if !ok {
		return
	}

	user, ok := app.validateJellyfinCredentials(username, password, gc, true)
	if !ok {
		return
	}

	token, refresh, err := CreateToken(user.ID, user.ID, false)
	if err != nil {
		app.err.Printf("getUserToken failed: Couldn't generate user token (%s)", err)
		respond(500, "Couldn't generate user token", gc)
		return
	}

	app.debug.Printf("Token generated for non-admin user \"%s\"", username)
	uri := "/my"
	if strings.HasPrefix(gc.Request.RequestURI, app.URLBase) {
		uri = "/accounts/my"
	}
	gc.SetCookie("user-refresh", refresh, REFRESH_TOKEN_VALIDITY_SEC, uri, gc.Request.URL.Hostname(), true, true)
	gc.JSON(200, getTokenDTO{token})
}

// @Summary Grabs an user-access token using a refresh token from cookies.
// @Produce json
// @Success 200 {object} getTokenDTO
// @Failure 401 {object} stringResponse
// @Router /my/token/refresh [get]
// @tags Auth
func (app *appContext) getUserTokenRefresh(gc *gin.Context) {
	jellyfinLogin := app.config.Section("ui").Key("jellyfin_login").MustBool(true)
	if !jellyfinLogin {
		app.err.Println("Enable Jellyfin Login to use the User Page feature.")
		respond(500, "Contact Admin", gc)
		return
	}

	app.logIpInfo(gc, true, "UserToken request (refresh token)")
	claims, ok := app.decodeValidateRefreshCookie(gc, "user-refresh")
	if !ok {
		return
	}

	jfID := claims["jfid"].(string)

	jwt, refresh, err := CreateToken(jfID, jfID, false)
	if err != nil {
		app.err.Printf("getUserToken failed: Couldn't generate user token (%s)", err)
		respond(500, "Couldn't generate user token", gc)
		return
	}

	gc.SetCookie("user-refresh", refresh, REFRESH_TOKEN_VALIDITY_SEC, "/my", gc.Request.URL.Hostname(), true, true)
	gc.JSON(200, getTokenDTO{jwt})
}
