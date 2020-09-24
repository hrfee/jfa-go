package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v3"
)

func (app *appContext) webAuth() gin.HandlerFunc {
	return app.authenticate
}

// CreateToken returns a web token as well as a refresh token, which can be used to obtain new tokens.
func CreateToken(userId, jfId string) (string, string, error) {
	var token, refresh string
	claims := jwt.MapClaims{
		"valid": true,
		"id":    userId,
		"exp":   strconv.FormatInt(time.Now().Add(time.Minute*20).Unix(), 10),
		"jfid":  jfId,
		"type":  "bearer",
	}

	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tk.SignedString([]byte(os.Getenv("JFA_SECRET")))
	if err != nil {
		return "", "", err
	}
	claims["exp"] = strconv.FormatInt(time.Now().Add(time.Hour*24).Unix(), 10)
	claims["type"] = "refresh"
	tk = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refresh, err = tk.SignedString([]byte(os.Getenv("JFA_SECRET")))
	if err != nil {
		return "", "", err
	}
	return token, refresh, nil
}

// Check header for token
func (app *appContext) authenticate(gc *gin.Context) {
	header := strings.SplitN(gc.Request.Header.Get("Authorization"), " ", 2)
	if header[0] != "Basic" {
		app.debug.Println("Invalid authentication header")
		respond(401, "Unauthorized", gc)
		return
	}
	auth, _ := base64.StdEncoding.DecodeString(header[1])
	creds := strings.SplitN(string(auth), ":", 2)
	token, err := jwt.Parse(creds[0], checkToken)
	if err != nil {
		app.debug.Printf("Auth denied: %s", err)
		respond(401, "Unauthorized", gc)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	expiryUnix, err := strconv.ParseInt(claims["exp"].(string), 10, 64)
	if err != nil {
		app.debug.Printf("Auth denied: %s", err)
		respond(401, "Unauthorized", gc)
		return
	}
	expiry := time.Unix(expiryUnix, 0)
	if !(ok && token.Valid && claims["type"].(string) == "bearer" && expiry.After(time.Now())) {
		app.debug.Printf("Auth denied: Invalid token")
		respond(401, "Unauthorized", gc)
		return
	}
	userID := claims["id"].(string)
	jfID := claims["jfid"].(string)
	match := false
	for _, user := range app.users {
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
	app.debug.Println("Auth succeeded")
	gc.Next()
}

func checkToken(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("Unexpected signing method %v", token.Header["alg"])
	}
	return []byte(os.Getenv("JFA_SECRET")), nil
}

// getToken checks the header for a username and password, as well as checking the refresh cookie.
func (app *appContext) getToken(gc *gin.Context) {
	app.info.Println("Token requested (login attempt)")
	header := strings.SplitN(gc.Request.Header.Get("Authorization"), " ", 2)
	auth, _ := base64.StdEncoding.DecodeString(header[1])
	creds := strings.SplitN(string(auth), ":", 2)
	// check cookie first
	var userID, jfID string
	valid := false
	noLogin := false
	checkLogin := func() {
		if creds[0] == "" || creds[1] == "" {
			app.debug.Println("Auth denied: blank username/password")
			respond(401, "Unauthorized", gc)
			return
		}
		match := false
		for _, user := range app.users {
			if user.Username == creds[0] && user.Password == creds[1] {
				match = true
				app.debug.Println("Found existing user")
				userID = user.UserID
				break
			}
		}
		if !app.jellyfinLogin && !match {
			app.info.Println("Auth denied: Invalid username/password")
			respond(401, "Unauthorized", gc)
			return
		}
		if !match {
			var status int
			var err error
			var user map[string]interface{}
			user, status, err = app.authJf.authenticate(creds[0], creds[1])
			if status != 200 || err != nil {
				if status == 401 || status == 400 {
					app.info.Println("Auth denied: Invalid username/password (Jellyfin)")
					respond(401, "Unauthorized", gc)
					return
				}
				app.err.Printf("Auth failed: Couldn't authenticate with Jellyfin (%d/%s)", status, err)
				respond(500, "Jellyfin error", gc)
				return
			}
			jfID = user["Id"].(string)
			if app.config.Section("ui").Key("admin_only").MustBool(true) {
				if !user["Policy"].(map[string]interface{})["IsAdministrator"].(bool) {
					app.debug.Printf("Auth denied: Users \"%s\" isn't admin", creds[0])
					respond(401, "Unauthorized", gc)
					return
				}
			}
			// New users are only added when using jellyfinLogin.
			userID = shortuuid.New()
			newUser := User{
				UserID: userID,
			}
			app.debug.Printf("Token generated for user \"%s\"", creds[0])
			app.users = append(app.users, newUser)
		}
		valid = true
	}
	checkCookie := func() {
		cookie, err := gc.Cookie("refresh")
		if err == nil && cookie != "" {
			for _, token := range app.invalidTokens {
				if cookie == token {
					if creds[0] == "" || creds[1] == "" {
						app.debug.Println("getToken denied: Invalid refresh token and no username/password provided")
						respond(401, "Unauthorized", gc)
						noLogin = true
						return
					}
					app.debug.Println("getToken: Invalid token but username/password provided")
					return
				}
			}
			token, err := jwt.Parse(cookie, checkToken)
			if err != nil {
				if creds[0] == "" || creds[1] == "" {
					app.debug.Println("getToken denied: Invalid refresh token and no username/password provided")
					respond(401, "Unauthorized", gc)
					noLogin = true
					return
				}
				app.debug.Println("getToken: Invalid token but username/password provided")
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			expiryUnix, err := strconv.ParseInt(claims["exp"].(string), 10, 64)
			if err != nil {
				if creds[0] == "" || creds[1] == "" {
					app.debug.Printf("getToken denied: Invalid token (%s) and no username/password provided", err)
					respond(401, "Unauthorized", gc)
					noLogin = true
					return
				}
				app.debug.Printf("getToken: Invalid token (%s) but username/password provided", err)
				return
			}
			expiry := time.Unix(expiryUnix, 0)
			if !(ok && token.Valid && claims["type"].(string) == "refresh" && expiry.After(time.Now())) {
				if creds[0] == "" || creds[1] == "" {
					app.debug.Printf("getToken denied: Invalid token (%s) and no username/password provided", err)
					respond(401, "Unauthorized", gc)
					noLogin = true
					return
				}
				app.debug.Printf("getToken: Invalid token (%s) but username/password provided", err)
				return
			}
			userID = claims["id"].(string)
			jfID = claims["jfid"].(string)
			valid = true
		}
	}
	checkCookie()
	if !valid && !noLogin {
		checkLogin()
	}
	if valid {
		token, refresh, err := CreateToken(userID, jfID)
		if err != nil {
			app.err.Printf("getToken failed: Couldn't generate token (%s)", err)
			respond(500, "Couldn't generate token", gc)
			return
		}
		gc.SetCookie("refresh", refresh, (3600 * 24), "/", gc.Request.URL.Hostname(), true, true)
		gc.JSON(200, map[string]string{"token": token})
	} else {
		gc.AbortWithStatus(401)
	}
}
