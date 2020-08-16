package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v3"
	"gopkg.in/ini.v1"
)

// Username is JWT!
type User struct {
	UserID   string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type appContext struct {
	// defaults         *Config
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
	quit             chan os.Signal
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
	app := new(appContext)
	userConfigDir, _ := os.UserConfigDir()
	app.data_path = filepath.Join(userConfigDir, "jfa-go")
	app.config_path = filepath.Join(app.data_path, "config.ini")
	executable, _ := os.Executable()
	app.local_path = filepath.Join(filepath.Dir(executable), "data")

	app.info = log.New(os.Stdout, "[INFO] ", log.Ltime)
	app.err = log.New(os.Stdout, "[ERROR] ", log.Ltime|log.Lshortfile)

	dataPath := flag.String("data", app.data_path, "alternate path to data directory.")
	configPath := flag.String("config", app.config_path, "alternate path to config file.")
	host := flag.String("host", "", "alternate address to host web ui on.")
	port := flag.Int("port", 0, "alternate port to host web ui on.")

	flag.Parse()
	if app.config_path == *configPath && app.data_path != *dataPath {
		app.data_path = *dataPath
		app.config_path = filepath.Join(app.data_path, "config.ini")
	} else if app.config_path != *configPath && app.data_path == *dataPath {
		app.config_path = *configPath
	} else {
		app.config_path = *configPath
		app.data_path = *dataPath
	}

	// Env variables are necessary because syscall.Exec for self-restarts doesn't doesn't work with arguments for some reason.

	if v := os.Getenv("JFA_CONFIGPATH"); v != "" {
		app.config_path = v
	}
	if v := os.Getenv("JFA_DATAPATH"); v != "" {
		app.data_path = v
	}

	os.Setenv("JFA_CONFIGPATH", app.config_path)
	os.Setenv("JFA_DATAPATH", app.data_path)

	var firstRun bool
	if _, err := os.Stat(app.data_path); os.IsNotExist(err) {
		os.Mkdir(app.data_path, 0700)
	}
	if _, err := os.Stat(app.config_path); os.IsNotExist(err) {
		firstRun = true
		dConfigPath := filepath.Join(app.local_path, "config-default.ini")
		var dConfig *os.File
		dConfig, err = os.Open(dConfigPath)
		if err != nil {
			app.err.Fatalf("Couldn't find default config file \"%s\"", dConfigPath)
		}
		defer dConfig.Close()
		var nConfig *os.File
		nConfig, err := os.Create(app.config_path)
		if err != nil {
			app.err.Fatalf("Couldn't open config file for writing: \"%s\"", app.config_path)
		}
		defer nConfig.Close()
		_, err = io.Copy(nConfig, dConfig)
		if err != nil {
			app.err.Fatalf("Couldn't copy default config. To do this manually, copy\n%s\nto\n%s", dConfigPath, app.config_path)
		}
		app.info.Printf("Copied default configuration to \"%s\"", app.config_path)
	}
	var debugMode bool
	var address string
	if app.loadConfig() != nil {
		app.err.Fatalf("Failed to load config file \"%s\"", app.config_path)
	}
	app.version = app.config.Section("jellyfin").Key("version").String()

	debugMode = app.config.Section("ui").Key("debug").MustBool(true)
	if debugMode {
		app.debug = log.New(os.Stdout, "[DEBUG] ", log.Ltime|log.Lshortfile)
	} else {
		app.debug = log.New(ioutil.Discard, "", 0)
	}

	if !firstRun {
		app.host = app.config.Section("ui").Key("host").String()
		app.port = app.config.Section("ui").Key("port").MustInt(8056)

		if *host != app.host && *host != "" {
			app.host = *host
		}
		if *port != app.port && *port > 0 {
			app.port = *port
		}

		if h := os.Getenv("JFA_HOST"); h != "" {
			app.host = h
			if p := os.Getenv("JFA_PORT"); p != "" {
				var port int
				_, err := fmt.Sscan(p, &port)
				if err == nil {
					app.port = port
				}
			}
		}

		address = fmt.Sprintf("%s:%d", app.host, app.port)

		app.debug.Printf("Loaded config file \"%s\"", app.config_path)

		if app.config.Section("ui").Key("bs5").MustBool(false) {
			app.cssFile = "bs5-jf.css"
			app.bsVersion = 5
		} else {
			app.cssFile = "bs4-jf.css"
			app.bsVersion = 4
		}

		app.debug.Println("Loading storage")

		app.storage.invite_path = filepath.Join(app.data_path, "invites.json")
		app.storage.loadInvites()
		app.storage.emails_path = filepath.Join(app.data_path, "emails.json")
		app.storage.loadEmails()
		app.storage.policy_path = filepath.Join(app.data_path, "user_template.json")
		app.storage.loadPolicy()
		app.storage.configuration_path = filepath.Join(app.data_path, "user_configuration.json")
		app.storage.loadConfiguration()
		app.storage.displayprefs_path = filepath.Join(app.data_path, "user_displayprefs.json")
		app.storage.loadDisplayprefs()

		app.configBase_path = filepath.Join(app.local_path, "config-base.json")
		config_base, _ := ioutil.ReadFile(app.configBase_path)
		json.Unmarshal(config_base, &app.configBase)

		themes := map[string]string{
			"Jellyfin (Dark)":   fmt.Sprintf("bs%d-jf.css", app.bsVersion),
			"Bootstrap (Light)": fmt.Sprintf("bs%d.css", app.bsVersion),
			"Custom CSS":        "",
		}
		if val, ok := themes[app.config.Section("ui").Key("theme").String()]; ok {
			app.cssFile = val
		}
		app.debug.Printf("Using css file \"%s\"", app.cssFile)
		secret, err := GenerateSecret(16)
		if err != nil {
			app.err.Fatal(err)
		}
		os.Setenv("JFA_SECRET", secret)
		app.jellyfinLogin = true
		if val, _ := app.config.Section("ui").Key("jellyfin_login").Bool(); !val {
			app.jellyfinLogin = false
			user := User{}
			user.UserID = shortuuid.New()
			user.Username = app.config.Section("ui").Key("username").String()
			user.Password = app.config.Section("ui").Key("password").String()
			app.users = append(app.users, user)
		} else {
			app.debug.Println("Using Jellyfin for authentication")
		}

		server := app.config.Section("jellyfin").Key("server").String()
		app.jf.init(server, "jfa-go", app.version, "hrfee-arch", "hrfee-arch")
		var status int
		_, status, err = app.jf.authenticate(app.config.Section("jellyfin").Key("username").String(), app.config.Section("jellyfin").Key("password").String())
		if status != 200 || err != nil {
			app.err.Fatalf("Failed to authenticate with Jellyfin @ %s: Code %d", server, status)
		}
		app.info.Printf("Authenticated with %s", server)
		app.authJf.init(server, "jfa-go", app.version, "auth", "auth")

		app.loadStrftime()

		validatorConf := ValidatorConf{
			"characters":           app.config.Section("password_validation").Key("min_length").MustInt(0),
			"uppercase characters": app.config.Section("password_validation").Key("upper").MustInt(0),
			"lowercase characters": app.config.Section("password_validation").Key("lower").MustInt(0),
			"numbers":              app.config.Section("password_validation").Key("number").MustInt(0),
			"special characters":   app.config.Section("password_validation").Key("special").MustInt(0),
		}
		if !app.config.Section("password_validation").Key("enabled").MustBool(false) {
			for key := range validatorConf {
				validatorConf[key] = 0
			}
		}
		app.validator.init(validatorConf)

		app.email.init(app)

		inviteDaemon := NewRepeater(time.Duration(60*time.Second), app)
		go inviteDaemon.Run()

		if app.config.Section("password_resets").Key("enabled").MustBool(false) {
			go app.StartPWR()
		}
	} else {
		debugMode = false
		gin.SetMode(gin.ReleaseMode)
		address = "0.0.0.0:8056"
	}

	app.info.Println("Loading routes")
	router := gin.New()

	setGinLogger(router, debugMode)

	router.Use(gin.Recovery())
	router.Use(static.Serve("/", static.LocalFile(filepath.Join(app.local_path, "static"), false)))
	router.LoadHTMLGlob(filepath.Join(app.local_path, "templates", "*"))
	router.NoRoute(app.NoRouteHandler)
	if debugMode {
		app.debug.Println("Loading pprof")
		pprof.Register(router)
	}
	if !firstRun {
		router.GET("/", app.AdminPage)
		router.GET("/getToken", app.GetToken)
		router.POST("/newUser", app.NewUser)
		router.Use(static.Serve("/invite/", static.LocalFile(filepath.Join(app.local_path, "static"), false)))
		router.GET("/invite/:invCode", app.InviteProxy)
		api := router.Group("/", app.webAuth())
		api.POST("/generateInvite", app.GenerateInvite)
		api.GET("/getInvites", app.GetInvites)
		api.POST("/setNotify", app.SetNotify)
		api.POST("/deleteInvite", app.DeleteInvite)
		api.GET("/getUsers", app.GetUsers)
		api.POST("/modifyUsers", app.ModifyEmails)
		api.POST("/setDefaults", app.SetDefaults)
		api.GET("/getConfig", app.GetConfig)
		api.POST("/modifyConfig", app.ModifyConfig)
		app.info.Printf("Starting router @ %s", address)
	} else {
		router.GET("/", func(gc *gin.Context) {
			gc.HTML(200, "setup.html", gin.H{})
		})
		router.POST("/testJF", app.TestJF)
		router.POST("/modifyConfig", app.ModifyConfig)
		app.info.Printf("Loading setup @ %s", address)
	}

	srv := &http.Server{
		Addr:    address,
		Handler: router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			app.err.Printf("Failure serving: %s", err)
		}
	}()
	app.quit = make(chan os.Signal)
	signal.Notify(app.quit, os.Interrupt)
	<-app.quit
	app.info.Println("Shutting down...")

	cntx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := srv.Shutdown(cntx); err != nil {
		app.err.Fatalf("Server shutdown error: %s", err)
	}
}
