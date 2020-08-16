package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v3"
)

func (app *appContext) webAuth() gin.HandlerFunc {
	return app.authenticate
}

func (app *appContext) authenticate(gc *gin.Context) {
	header := strings.SplitN(gc.Request.Header.Get("Authorization"), " ", 2)
	if header[0] != "Basic" {
		app.debug.Println("Invalid authentication header")
		respond(401, "Unauthorized", gc)
		return
	}
	auth, _ := base64.StdEncoding.DecodeString(header[1])
	creds := strings.SplitN(string(auth), ":", 2)
	token, err := jwt.Parse(creds[0], func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			app.debug.Printf("Invalid JWT signing method %s", token.Header["alg"])
			return nil, fmt.Errorf("Unexpected signing method %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JFA_SECRET")), nil
	})
	if err != nil {
		app.debug.Printf("Auth denied: %s", err)
		respond(401, "Unauthorized", gc)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	var userId string
	var jfId string
	if ok && token.Valid {
		userId = claims["id"].(string)
		jfId = claims["jfid"].(string)
	} else {
		app.debug.Printf("Invalid token")
		respond(401, "Unauthorized", gc)
		return
	}
	match := false
	for _, user := range app.users {
		if user.UserID == userId {
			match = true
		}
	}
	if !match {
		app.debug.Printf("Couldn't find user ID %s", userId)
		respond(401, "Unauthorized", gc)
		return
	}
	gc.Set("jfId", jfId)
	gc.Set("userId", userId)
	app.debug.Println("Authentication successful")
	gc.Next()
}

func (app *appContext) GetToken(gc *gin.Context) {
	app.info.Println("Token requested (login attempt)")
	header := strings.SplitN(gc.Request.Header.Get("Authorization"), " ", 2)
	if header[0] != "Basic" {
		app.debug.Println("Invalid authentication header")
		respond(401, "Unauthorized", gc)
		return
	}
	auth, _ := base64.StdEncoding.DecodeString(header[1])
	creds := strings.SplitN(string(auth), ":", 2)
	match := false
	var userId string
	for _, user := range app.users {
		if user.Username == creds[0] && user.Password == creds[1] {
			match = true
			userId = user.UserID
		}
	}
	jfId := ""
	if !match {
		if !app.jellyfinLogin {
			app.info.Println("Auth failed: Invalid username and/or password")
			respond(401, "Unauthorized", gc)
			return
		}
		var status int
		var err error
		var user map[string]interface{}
		user, status, err = app.authJf.authenticate(creds[0], creds[1])
		jfId = user["Id"].(string)
		if status != 200 || err != nil {
			if status == 401 {
				app.info.Println("Auth failed: Invalid username and/or password")
				respond(401, "Unauthorized", gc)
				return
			}
			app.err.Printf("Auth failed: Couldn't authenticate with Jellyfin: Code %d", status)
			respond(500, "Jellyfin error", gc)
			return
		} else {
			if app.config.Section("ui").Key("admin_only").MustBool(true) {
				if !user["Policy"].(map[string]interface{})["IsAdministrator"].(bool) {
					app.debug.Printf("Auth failed: User \"%s\" isn't admin", creds[0])
					respond(401, "Unauthorized", gc)
				}
			}
			newuser := User{}
			newuser.UserID = shortuuid.New()
			userId = newuser.UserID
			// uuid, nothing else identifiable!
			app.debug.Printf("Token generated for user \"%s\"", creds[0])
			app.users = append(app.users, newuser)
		}
	}
	token, err := CreateToken(userId, jfId)
	if err != nil {
		respond(500, "Error generating token", gc)
	}
	resp := map[string]string{"token": token}
	gc.JSON(200, resp)
}

func CreateToken(userId string, jfId string) (string, error) {
	claims := jwt.MapClaims{
		"valid": true,
		"id":    userId,
		"exp":   time.Now().Add(time.Minute * 20).Unix(),
		"jfid":  jfId,
	}

	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tk.SignedString([]byte(os.Getenv("JFA_SECRET")))
	if err != nil {
		return "", err
	}
	return token, nil
}

func respond(code int, message string, gc *gin.Context) {
	resp := map[string]string{"error": message}
	gc.JSON(code, resp)
	gc.Abort()
}
