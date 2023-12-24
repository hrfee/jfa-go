package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/hrfee/mediabrowser"
	"github.com/lithammer/shortuuid/v3"
)

const (
	TOKEN_VALIDITY_SEC         = 20 * 60
	REFRESH_TOKEN_VALIDITY_SEC = 3600 * 24
)

func (app *appContext) logIpInfo(gc *gin.Context, user bool, out string) {
	if (user && LOGIPU) || (!user && LOGIP) {
		out += fmt.Sprintf(" (ip=%s)", gc.ClientIP())
	}
	app.info.Println(out)
}
func (app *appContext) logIpDebug(gc *gin.Context, user bool, out string) {
	if (user && LOGIPU) || (!user && LOGIP) {
		out += fmt.Sprintf(" (ip=%s)", gc.ClientIP())
	}
	app.debug.Println(out)
}
func (app *appContext) logIpErr(gc *gin.Context, user bool, out string) {
	if (user && LOGIPU) || (!user && LOGIP) {
		out += fmt.Sprintf(" (ip=%s)", gc.ClientIP())
	}
	app.err.Println(out)
}

func (app *appContext) webAuth() gin.HandlerFunc {
	return app.authenticate
}

// CreateToken returns a web token as well as a refresh token, which can be used to obtain new tokens.
func CreateToken(userId, jfId string, admin bool) (string, string, error) {
	var token, refresh string
	claims := jwt.MapClaims{
		"valid": true,
		"id":    userId,
		"exp":   time.Now().Add(time.Second * TOKEN_VALIDITY_SEC).Unix(),
		"jfid":  jfId,
		"admin": admin,
		"type":  "bearer",
	}
	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tk.SignedString([]byte(os.Getenv("JFA_SECRET")))
	if err != nil {
		return "", "", err
	}
	claims["exp"] = time.Now().Add(time.Second * REFRESH_TOKEN_VALIDITY_SEC).Unix()
	claims["type"] = "refresh"
	tk = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refresh, err = tk.SignedString([]byte(os.Getenv("JFA_SECRET")))
	if err != nil {
		return "", "", err
	}
	return token, refresh, nil
}

// Caller should return if this returns false.
func (app *appContext) decodeValidateAuthHeader(gc *gin.Context) (claims jwt.MapClaims, ok bool) {
	ok = false
	header := strings.SplitN(gc.Request.Header.Get("Authorization"), " ", 2)
	if header[0] != "Bearer" {
		app.debug.Println("Invalid authorization header")
		respond(401, "Unauthorized", gc)
		return
	}
	token, err := jwt.Parse(string(header[1]), checkToken)
	if err != nil {
		app.debug.Printf("Auth denied: %s", err)
		respond(401, "Unauthorized", gc)
		return
	}
	claims, ok = token.Claims.(jwt.MapClaims)
	if !ok {
		app.debug.Println("Invalid JWT")
		respond(401, "Unauthorized", gc)
		return
	}
	expiryUnix := int64(claims["exp"].(float64))
	if err != nil {
		app.debug.Printf("Auth denied: %s", err)
		respond(401, "Unauthorized", gc)
		ok = false
		return
	}
	expiry := time.Unix(expiryUnix, 0)
	if !(ok && token.Valid && claims["type"].(string) == "bearer" && expiry.After(time.Now())) {
		app.debug.Printf("Auth denied: Invalid token")
		// app.debug.Printf("Expiry: %+v, OK: %t, Valid: %t, ClaimType: %s\n", expiry, ok, token.Valid, claims["type"].(string))
		respond(401, "Unauthorized", gc)
		ok = false
		return
	}
	ok = true
	return
}

// Check header for token
func (app *appContext) authenticate(gc *gin.Context) {
	claims, ok := app.decodeValidateAuthHeader(gc)
	if !ok {
		return
	}
	isAdminToken := claims["admin"].(bool)
	if !isAdminToken {
		app.debug.Printf("Auth denied: Token was not for admin access")
		respond(401, "Unauthorized", gc)
		return
	}

	userID := claims["id"].(string)
	jfID := claims["jfid"].(string)
	match := false
	for _, user := range app.adminUsers {
		if user.UserID == userID {
			match = true
			break
		}
	}
	if !match {
		app.debug.Printf("Couldn't find user ID \"%s\"", userID)
		respond(401, "Unauthorized", gc)
		return
	}
	gc.Set("jfId", jfID)
	gc.Set("userId", userID)
	gc.Set("userMode", false)
	app.debug.Println("Auth succeeded")
	gc.Next()
}

func checkToken(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("Unexpected signing method %v", token.Header["alg"])
	}
	return []byte(os.Getenv("JFA_SECRET")), nil
}

type getTokenDTO struct {
	Token string `json:"token" example:"kjsdklsfdkljfsjsdfklsdfkldsfjdfskjsdfjklsdf"` // API token for use with everything else.
}

func (app *appContext) decodeValidateLoginHeader(gc *gin.Context, userpage bool) (username, password string, ok bool) {
	header := strings.SplitN(gc.Request.Header.Get("Authorization"), " ", 2)
	auth, _ := base64.StdEncoding.DecodeString(header[1])
	creds := strings.SplitN(string(auth), ":", 2)
	username = creds[0]
	password = creds[1]
	ok = false
	if username == "" || password == "" {
		app.logIpDebug(gc, userpage, "Auth denied: blank username/password")
		respond(401, "Unauthorized", gc)
		return
	}
	ok = true
	return
}

func (app *appContext) validateJellyfinCredentials(username, password string, gc *gin.Context, userpage bool) (user mediabrowser.User, ok bool) {
	ok = false
	user, status, err := app.authJf.Authenticate(username, password)
	if status != 200 || err != nil {
		if status == 401 || status == 400 {
			app.logIpInfo(gc, userpage, "Auth denied: Invalid username/password (Jellyfin)")
			respond(401, "Unauthorized", gc)
			return
		}
		if status == 403 {
			app.logIpInfo(gc, userpage, "Auth denied: Jellyfin account disabled")
			respond(403, "yourAccountWasDisabled", gc)
			return
		}
		app.err.Printf("Auth failed: Couldn't authenticate with Jellyfin (%d/%s)", status, err)
		respond(500, "Jellyfin error", gc)
		return
	}
	ok = true
	return
}

// @Summary Grabs an API token using username & password.
// @description If viewing docs locally, click the lock icon next to this, login with your normal jfa-go credentials. Click 'try it out', then 'execute' and an API Key will be returned, copy it (not including quotes). On any of the other routes, click the lock icon and set the API key as "Bearer `your api key`".
// @Produce json
// @Success 200 {object} getTokenDTO
// @Failure 401 {object} stringResponse
// @Router /token/login [get]
// @tags Auth
// @Security getTokenAuth
func (app *appContext) getTokenLogin(gc *gin.Context) {
	app.logIpInfo(gc, false, "Token requested (login attempt)")
	username, password, ok := app.decodeValidateLoginHeader(gc, false)
	if !ok {
		return
	}
	var userID, jfID string
	match := false
	for _, user := range app.adminUsers {
		if user.Username == username && user.Password == password {
			match = true
			app.debug.Println("Found existing user")
			userID = user.UserID
			break
		}
	}
	if !app.jellyfinLogin && !match {
		app.logIpInfo(gc, false, "Auth denied: Invalid username/password")
		respond(401, "Unauthorized", gc)
		return
	}
	if !match {
		user, ok := app.validateJellyfinCredentials(username, password, gc, false)
		if !ok {
			return
		}
		jfID = user.ID
		if !app.config.Section("ui").Key("allow_all").MustBool(false) {
			accountsAdmin := false
			adminOnly := app.config.Section("ui").Key("admin_only").MustBool(true)
			if emailStore, ok := app.storage.GetEmailsKey(jfID); ok {
				accountsAdmin = emailStore.Admin
			}
			accountsAdmin = accountsAdmin || (adminOnly && user.Policy.IsAdministrator)
			if !accountsAdmin {
				app.debug.Printf("Auth denied: Users \"%s\" isn't admin", username)
				respond(401, "Unauthorized", gc)
				return
			}
		}
		// New users are only added when using jellyfinLogin.
		userID = shortuuid.New()
		newUser := User{
			UserID: userID,
		}
		app.debug.Printf("Token generated for user \"%s\"", username)
		app.adminUsers = append(app.adminUsers, newUser)
	}
	token, refresh, err := CreateToken(userID, jfID, true)
	if err != nil {
		app.err.Printf("getToken failed: Couldn't generate token (%s)", err)
		respond(500, "Couldn't generate token", gc)
		return
	}
	host := gc.Request.URL.Hostname()
	gc.SetCookie("refresh", refresh, REFRESH_TOKEN_VALIDITY_SEC, "/", host, true, true)
	gc.JSON(200, getTokenDTO{token})
}

func (app *appContext) decodeValidateRefreshCookie(gc *gin.Context, cookieName string) (claims jwt.MapClaims, ok bool) {
	ok = false
	cookie, err := gc.Cookie(cookieName)
	if err != nil || cookie == "" {
		app.debug.Printf("getTokenRefresh denied: Couldn't get token: %s", err)
		respond(400, "Couldn't get token", gc)
		return
	}
	for _, token := range app.invalidTokens {
		if cookie == token {
			app.debug.Println("getTokenRefresh: Invalid token")
			respond(401, "Invalid token", gc)
			return
		}
	}
	token, err := jwt.Parse(cookie, checkToken)
	if err != nil {
		app.debug.Println("getTokenRefresh: Invalid token")
		respond(400, "Invalid token", gc)
		return
	}
	claims, ok = token.Claims.(jwt.MapClaims)
	expiryUnix := int64(claims["exp"].(float64))
	if err != nil {
		app.debug.Printf("getTokenRefresh: Invalid token expiry: %s", err)
		respond(401, "Invalid token", gc)
		ok = false
		return
	}
	expiry := time.Unix(expiryUnix, 0)
	if !(ok && token.Valid && claims["type"].(string) == "refresh" && expiry.After(time.Now())) {
		app.debug.Printf("getTokenRefresh: Invalid token: %+v", err)
		respond(401, "Invalid token", gc)
		ok = false
		return
	}
	ok = true
	return
}

// @Summary Grabs an API token using a refresh token from cookies.
// @Produce json
// @Success 200 {object} getTokenDTO
// @Failure 401 {object} stringResponse
// @Router /token/refresh [get]
// @tags Auth
func (app *appContext) getTokenRefresh(gc *gin.Context) {
	app.logIpInfo(gc, false, "Token requested (refresh token)")
	claims, ok := app.decodeValidateRefreshCookie(gc, "refresh")
	if !ok {
		return
	}
	userID := claims["id"].(string)
	jfID := claims["jfid"].(string)
	jwt, refresh, err := CreateToken(userID, jfID, true)
	if err != nil {
		app.err.Printf("getTokenRefresh failed: Couldn't generate token (%s)", err)
		respond(500, "Couldn't generate token", gc)
		return
	}
	host := gc.Request.URL.Hostname()
	gc.SetCookie("refresh", refresh, REFRESH_TOKEN_VALIDITY_SEC, "/", host, true, true)
	gc.JSON(200, getTokenDTO{jwt})
}
