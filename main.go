package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v3"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Username is JWT!
type User struct {
	UserID   string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type appContext struct {
	config           *ini.File
	config_path      string
	configBase_path  string
	configBase       map[string]interface{}
	data_path        string
	local_path       string
	cssFile          string
	bsVersion        int
	jellyfinLogin    bool
	users            []User
	jf               Jellyfin
	authJf           Jellyfin
	datePattern      string
	timePattern      string
	storage          Storage
	validator        Validator
	email            Emailer
	info, debug, err *log.Logger
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

func setGinLogger(router *gin.Engine, debugMode bool) {
	if debugMode {
		router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[GIN/DEBUG] %s: %s(%s) => %d in %s; Errors: %s\n",
				param.TimeStamp.Format("15:04:05"),
				param.Method,
				param.Path,
				param.StatusCode,
				param.Latency,
				param.ErrorMessage,
			)
		}))
		gin.SetMode(gin.DebugMode)
	} else {
		router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[GIN] %s(%s) => %d\n",
				param.Method,
				param.Path,
				param.StatusCode,
			)
		}))
		gin.SetMode(gin.ReleaseMode)
	}
}

func main() {
	ctx := new(appContext)
	ctx.config_path = "/home/hrfee/.jf-accounts/config.ini"
	ctx.data_path = "/home/hrfee/.jf-accounts"
	ctx.local_path = "data"
	if ctx.loadConfig() != nil {
		ctx.err.Fatalf("Failed to load config file \"%s\"", ctx.config_path)
	}

	ctx.info = log.New(os.Stdout, "[INFO] ", log.Ltime)
	ctx.err = log.New(os.Stdout, "[ERROR] ", log.Ltime|log.Lshortfile)
	debugMode := ctx.config.Section("ui").Key("debug").MustBool(true)
	if debugMode {
		ctx.debug = log.New(os.Stdout, "[DEBUG] ", log.Ltime|log.Lshortfile)
	} else {
		ctx.debug = log.New(ioutil.Discard, "", 0)
	}

	ctx.debug.Printf("Loaded config file \"%s\"", ctx.config_path)

	if val, _ := ctx.config.Section("ui").Key("bs5").Bool(); val {
		ctx.cssFile = "bs5-jf.css"
		ctx.bsVersion = 5
	} else {
		ctx.cssFile = "bs4-jf.css"
		ctx.bsVersion = 4
	}
	// ctx.storage.formatter, _ = strftime.New("%Y-%m-%dT%H:%M:%S.%f")
	ctx.debug.Println("Loading storage")
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

	ctx.configBase_path = filepath.Join(ctx.local_path, "config-base.json")
	config_base, _ := ioutil.ReadFile(ctx.configBase_path)
	json.Unmarshal(config_base, &ctx.configBase)

	themes := map[string]string{
		"Jellyfin (Dark)":   fmt.Sprintf("bs%d-jf.css", ctx.bsVersion),
		"Bootstrap (Light)": fmt.Sprintf("bs%d.css", ctx.bsVersion),
		"Custom CSS":        "",
	}
	if val, ok := themes[ctx.config.Section("ui").Key("theme").String()]; ok {
		ctx.cssFile = val
	}
	ctx.debug.Printf("Using css file \"%s\"", ctx.cssFile)
	secret, err := GenerateSecret(16)
	if err != nil {
		ctx.err.Fatal(err)
	}
	os.Setenv("JFA_SECRET", secret)
	ctx.jellyfinLogin = true
	if val, _ := ctx.config.Section("ui").Key("jellyfin_login").Bool(); !val {
		ctx.jellyfinLogin = false
		user := User{}
		user.UserID = shortuuid.New()
		user.Username = ctx.config.Section("ui").Key("username").String()
		user.Password = ctx.config.Section("ui").Key("password").String()
		ctx.users = append(ctx.users, user)
	} else {
		ctx.debug.Println("Using Jellyfin for authentication")
	}

	server := ctx.config.Section("jellyfin").Key("server").String()
	ctx.jf.init(server, "jfa-go", "0.1", "hrfee-arch", "hrfee-arch")
	var status int
	_, status, err = ctx.jf.authenticate(ctx.config.Section("jellyfin").Key("username").String(), ctx.config.Section("jellyfin").Key("password").String())
	if status != 200 || err != nil {
		ctx.err.Fatalf("Failed to authenticate with Jellyfin @ %s: Code %d", server, status)
	}
	ctx.info.Printf("Authenticated with %s", server)
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
		for key := range validatorConf {
			validatorConf[key] = 0
		}
	}
	ctx.validator.init(validatorConf)

	ctx.email.init(ctx)

	inviteDaemon := NewRepeater(time.Duration(60*time.Second), ctx)
	go inviteDaemon.Run()

	if ctx.config.Section("password_resets").Key("enabled").MustBool(false) {
		ctx.StartPWR()
	}

	ctx.info.Println("Loading routes")
	router := gin.New()

	setGinLogger(router, debugMode)

	router.Use(gin.Recovery())
	router.Use(static.Serve("/", static.LocalFile("data/static", false)))
	router.Use(static.Serve("/invite/", static.LocalFile("data/static", false)))
	router.LoadHTMLGlob("data/templates/*")
	router.GET("/", ctx.AdminPage)
	router.GET("/getToken", ctx.GetToken)
	router.POST("/newUser", ctx.NewUser)
	router.GET("/invite/:invCode", ctx.InviteProxy)
	router.NoRoute(ctx.NoRouteHandler)
	api := router.Group("/", ctx.webAuth())
	api.POST("/generateInvite", ctx.GenerateInvite)
	api.GET("/getInvites", ctx.GetInvites)
	api.POST("/setNotify", ctx.SetNotify)
	api.POST("/deleteInvite", ctx.DeleteInvite)
	api.GET("/getUsers", ctx.GetUsers)
	api.POST("/modifyUsers", ctx.ModifyEmails)
	api.POST("/setDefaults", ctx.SetDefaults)
	api.GET("/getConfig", ctx.GetConfig)
	api.POST("/modifyConfig", ctx.ModifyConfig)
	addr := fmt.Sprintf("%s:%d", ctx.config.Section("ui").Key("host").String(), ctx.config.Section("ui").Key("port").MustInt(8056))
	ctx.info.Printf("Starting router @ %s", addr)
	router.Run(addr)
}
