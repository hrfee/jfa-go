package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v3"
	"gopkg.in/ini.v1"
	"io"
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
	host             string
	port             int
	version          string
}

func GenerateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), err
}

func setGinLogger(router *gin.Engine, debugMode bool) {
	if debugMode {
		router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[GIN/DEBUG] %s: %s(%s) => %d in %s; %s\n",
				param.TimeStamp.Format("15:04:05"),
				param.Method,
				param.Path,
				param.StatusCode,
				param.Latency,
				func() string {
					if param.ErrorMessage != "" {
						return "Error: " + param.ErrorMessage
					}
					return ""
				}(),
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
	userConfigDir, _ := os.UserConfigDir()
	ctx.data_path = filepath.Join(userConfigDir, "jfa-go")
	ctx.config_path = filepath.Join(ctx.data_path, "config.ini")
	ctx.local_path = "data"

	ctx.info = log.New(os.Stdout, "[INFO] ", log.Ltime)
	ctx.err = log.New(os.Stdout, "[ERROR] ", log.Ltime|log.Lshortfile)

	dataPath := flag.String("data", ctx.data_path, "alternate path to data directory.")
	configPath := flag.String("config", ctx.config_path, "alternate path to config file.")
	host := flag.String("host", "", "alternate address to host web ui on.")
	port := flag.Int("port", 0, "alternate port to host web ui on.")

	flag.Parse()
	fmt.Println(*dataPath, *configPath, *host, *port)
	if ctx.config_path == *configPath && ctx.data_path != *dataPath {
		ctx.config_path = filepath.Join(*dataPath, "config.ini")
	} else {
		ctx.config_path = *configPath
		ctx.data_path = *dataPath
	}

	// Env variables are necessary because syscall.Exec for self-restarts doesn't doesn't work with arguments for some reason.

	if v := os.Getenv("JFA_CONFIGPATH"); v != "" {
		ctx.config_path = v
	}
	if v := os.Getenv("JFA_DATAPATH"); v != "" {
		ctx.data_path = v
	}

	os.Setenv("JFA_CONFIGPATH", ctx.config_path)
	os.Setenv("JFA_DATAPATH", ctx.data_path)

	var firstRun bool
	if _, err := os.Stat(ctx.data_path); os.IsNotExist(err) {
		os.Mkdir(ctx.data_path, 0700)
	}
	if _, err := os.Stat(ctx.config_path); os.IsNotExist(err) {
		firstRun = true
		dConfigPath := filepath.Join(ctx.local_path, "config-default.ini")
		var dConfig *os.File
		dConfig, err = os.Open(dConfigPath)
		if err != nil {
			ctx.err.Fatalf("Couldn't find default config file \"%s\"", dConfigPath)
		}
		defer dConfig.Close()
		var nConfig *os.File
		nConfig, err := os.Create(ctx.config_path)
		if err != nil {
			ctx.err.Fatalf("Couldn't open config file for writing: \"%s\"", dConfigPath)
		}
		defer nConfig.Close()
		_, err = io.Copy(nConfig, dConfig)
		if err != nil {
			ctx.err.Fatalf("Couldn't copy default config. To do this manually, copy\n%s\nto\n%s", dConfigPath, ctx.config_path)
		}
		ctx.info.Printf("Copied default configuration to \"%s\"", ctx.config_path)
	}
	var debugMode bool
	var address string
	if ctx.loadConfig() != nil {
		ctx.err.Fatalf("Failed to load config file \"%s\"", ctx.config_path)
	}
	ctx.version = ctx.config.Section("jellyfin").Key("version").String()

	debugMode = ctx.config.Section("ui").Key("debug").MustBool(true)
	if debugMode {
		ctx.debug = log.New(os.Stdout, "[DEBUG] ", log.Ltime|log.Lshortfile)
	} else {
		ctx.debug = log.New(ioutil.Discard, "", 0)
	}

	if !firstRun {
		ctx.host = ctx.config.Section("ui").Key("host").String()
		ctx.port = ctx.config.Section("ui").Key("port").MustInt(8056)

		if *host != ctx.host && *host != "" {
			ctx.host = *host
		}
		if *port != ctx.port && *port > 0 {
			ctx.port = *port
		}

		if h := os.Getenv("JFA_HOST"); h != "" {
			ctx.host = h
			if p := os.Getenv("JFA_PORT"); p != "" {
				var port int
				_, err := fmt.Sscan(p, &port)
				if err == nil {
					ctx.port = port
				}
			}
		}

		address = fmt.Sprintf("%s:%d", ctx.host, ctx.port)

		ctx.debug.Printf("Loaded config file \"%s\"", ctx.config_path)

		if ctx.config.Section("ui").Key("bs5").MustBool(false) {
			ctx.cssFile = "bs5-jf.css"
			ctx.bsVersion = 5
		} else {
			ctx.cssFile = "bs4-jf.css"
			ctx.bsVersion = 4
		}

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
		ctx.jf.init(server, "jfa-go", ctx.version, "hrfee-arch", "hrfee-arch")
		var status int
		_, status, err = ctx.jf.authenticate(ctx.config.Section("jellyfin").Key("username").String(), ctx.config.Section("jellyfin").Key("password").String())
		if status != 200 || err != nil {
			ctx.err.Fatalf("Failed to authenticate with Jellyfin @ %s: Code %d", server, status)
		}
		ctx.info.Printf("Authenticated with %s", server)
		ctx.authJf.init(server, "jfa-go", ctx.version, "auth", "auth")

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
			go ctx.StartPWR()
		}
	} else {
		debugMode = false
		gin.SetMode(gin.ReleaseMode)
		address = "0.0.0.0:8056"
	}

	ctx.info.Println("Loading routes")
	router := gin.New()

	setGinLogger(router, debugMode)

	router.Use(gin.Recovery())
	router.Use(static.Serve("/", static.LocalFile("data/static", false)))
	router.LoadHTMLGlob("data/templates/*")
	router.NoRoute(ctx.NoRouteHandler)
	if !firstRun {
		router.GET("/", ctx.AdminPage)
		router.GET("/getToken", ctx.GetToken)
		router.POST("/newUser", ctx.NewUser)
		router.GET("/invite/:invCode", ctx.InviteProxy)
		router.Use(static.Serve("/invite/", static.LocalFile("data/static", false)))
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
		ctx.info.Printf("Starting router @ %s", address)
	} else {
		router.GET("/", func(gc *gin.Context) {
			gc.HTML(200, "setup.html", gin.H{})
		})
		router.POST("/testJF", ctx.TestJF)
		router.POST("/modifyConfig", ctx.ModifyConfig)
		ctx.info.Printf("Loading setup @ %s", address)
	}
	router.Run(address)
}
