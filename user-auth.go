package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
)

func (app *appContext) userAuth() gin.HandlerFunc {
	return app.userAuthenticate
}

func (app *appContext) userAuthenticate(gc *gin.Context) {
	jellyfinLogin := app.config.Section("ui").Key("jellyfin_login").MustBool(true)
	if !jellyfinLogin {
		app.err.Printf(lm.FailedAuthRequest, lm.UserPageRequiresJellyfinAuth)
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
		app.err.Printf(lm.FailedAuthRequest, lm.UserPageRequiresJellyfinAuth)
		respond(500, "Contact Admin", gc)
		return
	}
	app.logIpInfo(gc, true, fmt.Sprintf(lm.RequestingToken, lm.UserTokenLoginAttempt))
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
		app.err.Printf(lm.FailedGenerateToken, err)
		respond(500, "Couldn't generate user token", gc)
		return
	}

	// host := gc.Request.URL.Hostname()
	host := app.ExternalDomain
	uri := "/my"
	if strings.HasPrefix(gc.Request.RequestURI, app.URLBase) {
		uri = "/accounts/my"
	}
	gc.SetCookie("user-refresh", refresh, REFRESH_TOKEN_VALIDITY_SEC, uri, host, true, true)
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
		app.err.Printf(lm.FailedAuthRequest, lm.UserPageRequiresJellyfinAuth)
		respond(500, "Contact Admin", gc)
		return
	}

	app.logIpInfo(gc, true, fmt.Sprintf(lm.RequestingToken, lm.UserTokenRefresh))
	claims, ok := app.decodeValidateRefreshCookie(gc, "user-refresh")
	if !ok {
		return
	}

	jfID := claims["jfid"].(string)

	jwt, refresh, err := CreateToken(jfID, jfID, false)
	if err != nil {
		app.err.Printf(lm.FailedGenerateToken, err)
		respond(500, "Couldn't generate user token", gc)
		return
	}

	// host := gc.Request.URL.Hostname()
	host := app.ExternalDomain
	gc.SetCookie("user-refresh", refresh, REFRESH_TOKEN_VALIDITY_SEC, "/my", host, true, true)
	gc.JSON(200, getTokenDTO{jwt})
}
