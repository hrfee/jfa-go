package main

import (
	"encoding/base64"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"os"
	"strings"
	"time"
)

func (ctx *appContext) webAuth() gin.HandlerFunc {
	return ctx.authenticate
}

func (ctx *appContext) authenticate(gc *gin.Context) {
	for _, val := range ctx.users {
		fmt.Println("userid", val.UserID)
	}
	header := strings.SplitN(gc.Request.Header.Get("Authorization"), " ", 2)
	if header[0] != "Basic" {
		respond(401, "Unauthorized", gc)
		return
	}
	auth, _ := base64.StdEncoding.DecodeString(header[1])
	creds := strings.SplitN(string(auth), ":", 2)
	token, err := jwt.Parse(creds[0], func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JFA_SECRET")), nil
	})
	if err != nil {
		respond(401, "Unauthorized", gc)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	var userId uuid.UUID
	var jfId string
	if ok && token.Valid {
		userId, _ = uuid.Parse(claims["id"].(string))
		jfId = claims["jfid"].(string)
	} else {
		respond(401, "Unauthorized", gc)
		return
	}
	match := false
	for _, user := range ctx.users {
		fmt.Println("checking:", user.UserID, userId)
		if user.UserID == userId {
			match = true
		}
	}
	if !match {
		fmt.Println("error no match")
		respond(401, "Unauthorized", gc)
		return
	}
	gc.Set("jfId", jfId)
	gc.Set("userId", userId.String())
	gc.Next()
}

func (ctx *appContext) GetToken(gc *gin.Context) {
	header := strings.SplitN(gc.Request.Header.Get("Authorization"), " ", 2)
	if header[0] != "Basic" {
		respond(401, "Unauthorized", gc)
		return
	}
	auth, _ := base64.StdEncoding.DecodeString(header[1])
	creds := strings.SplitN(string(auth), ":", 2)
	match := false
	var userId uuid.UUID
	for _, user := range ctx.users {
		if user.Username == creds[0] && user.Password == creds[1] {
			match = true
			userId = user.UserID
		}
	}
	jfId := ""
	if !match {
		if !ctx.jellyfinLogin {
			respond(401, "Unauthorized", gc)
			return
		}
		// eventually, make authenticate return a user to avoid two calls.
		var status int
		var err error
		var user map[string]interface{}
		user, status, err = ctx.authJf.authenticate(creds[0], creds[1])
		jfId = user["Id"].(string)
		if status != 200 || err != nil {
			respond(401, "Unauthorized", gc)
			return
		} else {
			if ctx.config.Section("ui").Key("admin_only").MustBool(true) {
				if !user["Policy"].(map[string]interface{})["IsAdministrator"].(bool) {
					respond(401, "Unauthorized", gc)
				}
			}
			newuser := User{}
			newuser.UserID, _ = uuid.NewRandom()
			userId = newuser.UserID
			// uuid, nothing else identifiable!
			ctx.users = append(ctx.users, newuser)
		}
	}
	token, err := CreateToken(userId, jfId)
	if err != nil {
		respond(500, "Error generating token", gc)
	}
	resp := map[string]string{"token": token}
	gc.JSON(200, resp)
}

func CreateToken(userId uuid.UUID, jfId string) (string, error) {
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
