package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopkg.in/ini.v1"
	"os"
	"path/filepath"
)

// Username is JWT!
type User struct {
	UserID   uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Password string    `json:"password"`
}

type appContext struct {
	config        *ini.File
	config_path   string
	data_path     string
	local_path    string
	cssFile       string
	bsVersion     int
	jellyfinLogin bool
	users         []User
	jf            Jellyfin
	authJf        Jellyfin
	datePattern   string
	timePattern   string
	storage       Storage
	validator     Validator
}

func GenerateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), err
}

// func (ctx *Context) AdminJs(gc *gin.Context) {
// 	template, err := pongo2.FromFile("data/templates/admin.js")
// 	if err != nil {
// 		panic(err)
// 	}
// 	notifications, _ := ctx.config.Section("notifications").Key("enabled").Bool()
// 	out, err := template.Execute(pongo2.Context{
// 		"bsVersion":     ctx.bsVersion,
// 		"css_file":      ctx.cssFile,
// 		"notifications": notifications,
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	// bg.Ctx.Output.Header("Content-Type", "application/javascript")
// 	// bg.Ctx.WriteString(out)
// }

func main() {
	ctx := new(appContext)
	ctx.config_path = "/home/hrfee/.jf-accounts/config.ini"
	ctx.data_path = "/home/hrfee/.jf-accounts"
	ctx.local_path = "data"
	ctx.loadConfig()
	if val, _ := ctx.config.Section("ui").Key("bs5").Bool(); val {
		ctx.cssFile = "bs5-jf.css"
		ctx.bsVersion = 5
	} else {
		ctx.cssFile = "bs4-jf.css"
		ctx.bsVersion = 4
	}
	// ctx.storage.formatter, _ = strftime.New("%Y-%m-%dT%H:%M:%S.%f")
	ctx.storage.invite_path = filepath.Join(ctx.data_path, "invites.json")
	ctx.storage.loadInvites()
	ctx.storage.emails_path = filepath.Join(ctx.data_path, "emails.json")
	ctx.storage.loadEmails()
	ctx.storage.policy_path = filepath.Join(ctx.data_path, "user_template.json")
	ctx.storage.loadPolicy()
	ctx.storage.configuration_path = filepath.Join(ctx.data_path, "user_configuration.json")
	ctx.storage.loadConfiguration()
	ctx.storage.displayprefs_path = filepath.Join(ctx.data_path, "user_displayprefs.json")
	ctx.storage.loadDisplayprefs()

	themes := map[string]string{
		"Jellyfin (Dark)":   fmt.Sprintf("bs%d-jf.css", ctx.bsVersion),
		"Bootstrap (Light)": fmt.Sprintf("bs%d.css", ctx.bsVersion),
		"Custom CSS":        "",
	}
	if val, ok := themes[ctx.config.Section("ui").Key("theme").String()]; ok {
		ctx.cssFile = val
	}
	secret, err := GenerateSecret(16)
	if err != nil {
		panic(err)
	}
	os.Setenv("JFA_SECRET", secret)
	ctx.jellyfinLogin = true
	if val, _ := ctx.config.Section("ui").Key("jellyfin_login").Bool(); !val {
		ctx.jellyfinLogin = false
		user := User{}
		user.UserID, _ = uuid.NewUUID()
		user.Username = ctx.config.Section("ui").Key("username").String()
		user.Password = ctx.config.Section("ui").Key("password").String()
		ctx.users = append(ctx.users, user)
	}
	server := ctx.config.Section("jellyfin").Key("server").String()
	ctx.jf.init(server, "jfa-go", "0.1", "hrfee-arch", "hrfee-arch")
	ctx.jf.authenticate(ctx.config.Section("jellyfin").Key("username").String(), ctx.config.Section("jellyfin").Key("password").String())
	ctx.authJf.init(server, "jfa-go", "0.1", "auth", "auth")

	ctx.loadStrftime()

	validatorConf := ValidatorConf{
		"characters":           ctx.config.Section("password_validation").Key("min_length").MustInt(0),
		"uppercase characters": ctx.config.Section("password_validation").Key("upper").MustInt(0),
		"lowercase characters": ctx.config.Section("password_validation").Key("lower").MustInt(0),
		"numbers":              ctx.config.Section("password_validation").Key("number").MustInt(0),
		"special characters":   ctx.config.Section("password_validation").Key("special").MustInt(0),
	}

	if !ctx.config.Section("password_validation").Key("enabled").MustBool(false) {
		for key, _ := range validatorConf {
			validatorConf[key] = 0
		}
	}
	ctx.validator.init(validatorConf)

	router := gin.Default()
	router.Use(static.Serve("/", static.LocalFile("data/static", false)))
	router.LoadHTMLGlob("data/templates/*")
	router.GET("/", ctx.AdminPage)
	router.GET("/getToken", ctx.GetToken)
	api := router.Group("/", ctx.webAuth())
	api.POST("/generateInvite", ctx.GenerateInvite)
	api.GET("/getInvites", ctx.GetInvites)

	router.Run(":8080")
}
