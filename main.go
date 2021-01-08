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
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/common"
	_ "github.com/hrfee/jfa-go/docs"
	"github.com/hrfee/jfa-go/emby"
	"github.com/hrfee/jfa-go/jfapi"
	"github.com/hrfee/jfa-go/ombi"
	"github.com/lithammer/shortuuid/v3"
	"github.com/logrusorgru/aurora/v3"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gopkg.in/ini.v1"
)

// User is used for auth purposes.
type User struct {
	UserID   string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// contains everything the application needs, essentially. Wouldn't do this in the future.
type appContext struct {
	// defaults         *Config
	config         *ini.File
	configPath     string
	configBasePath string
	configBase     settings
	dataPath       string
	localPath      string
	cssClass       string
	jellyfinLogin  bool
	users          []User
	invalidTokens  []string
	// Keeping jf name because I can't think of a better one
	jf               common.MediaBrowserStruct
	authJf           common.MediaBrowserStruct
	ombi             *ombi.Ombi
	datePattern      string
	timePattern      string
	storage          Storage
	validator        Validator
	email            *Emailer
	info, debug, err *log.Logger
	host             string
	port             int
	version          string
	quit             chan os.Signal
	lang             Languages
	URLBase          string
}

// Languages stores the names and filenames of language files, and the index of that which is currently selected.
type Languages struct {
	langFiles   []os.FileInfo // Language filenames
	langOptions []string      // Language names
	chosenIndex int
}

func (app *appContext) loadHTML(router *gin.Engine) {
	customPath := app.config.Section("files").Key("html_templates").MustString("")
	templatePath := filepath.Join(app.localPath, "html")
	htmlFiles, err := ioutil.ReadDir(templatePath)
	if err != nil {
		app.err.Fatalf("Couldn't access template directory: \"%s\"", templatePath)
		return
	}
	loadFiles := make([]string, len(htmlFiles))
	for i, f := range htmlFiles {
		if _, err := os.Stat(filepath.Join(customPath, f.Name())); os.IsNotExist(err) {
			app.debug.Printf("Using default \"%s\"", f.Name())
			loadFiles[i] = filepath.Join(templatePath, f.Name())
		} else {
			app.info.Printf("Using custom \"%s\"", f.Name())
			loadFiles[i] = filepath.Join(filepath.Join(customPath, f.Name()))
		}
	}
	router.LoadHTMLFiles(loadFiles...)
}

func generateSecret(length int) (string, error) {
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
	} else {
		router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[GIN] %s(%s) => %d\n",
				param.Method,
				param.Path,
				param.StatusCode,
			)
		}))
	}
}

var (
	PLATFORM           string = runtime.GOOS
	SOCK               string = "jfa-go.sock"
	SRV                *http.Server
	RESTART            chan bool
	DATA, CONFIG, HOST *string
	PORT               *int
	DEBUG              *bool
	TEST               bool
	SWAGGER            *bool
)

func test(app *appContext) {
	fmt.Printf("\n\n----\n\n")
	settings := map[string]interface{}{
		"server":         app.jf.Server,
		"server version": app.jf.ServerInfo.Version,
		"server name":    app.jf.ServerInfo.Name,
		"authenticated?": app.jf.Authenticated,
		"access token":   app.jf.AccessToken,
		"username":       app.jf.Username,
	}
	for n, v := range settings {
		fmt.Println(n, ":", v)
	}
	users, status, err := app.jf.GetUsers(false)
	fmt.Printf("GetUsers: code %d err %s maplength %d\n", status, err, len(users))
	fmt.Printf("View output? [y/n]: ")
	var choice string
	fmt.Scanln(&choice)
	if strings.Contains(choice, "y") {
		out, err := json.MarshalIndent(users, "", "  ")
		fmt.Print(string(out), err)
	}
	fmt.Printf("Enter a user to grab: ")
	var username string
	fmt.Scanln(&username)
	user, status, err := app.jf.UserByName(username, false)
	fmt.Printf("UserByName (%s): code %d err %s", username, status, err)
	out, _ := json.MarshalIndent(user, "", "  ")
	fmt.Print(string(out))
}

func start(asDaemon, firstCall bool) {
	// app encompasses essentially all useful functions.
	app := new(appContext)

	/*
		set default config, data and local paths
		also, confusing naming here. data_path is not the internal 'data' directory, rather the users .config/jfa-go folder.
		local_path is the internal 'data' directory.
	*/
	userConfigDir, _ := os.UserConfigDir()
	app.dataPath = filepath.Join(userConfigDir, "jfa-go")
	app.configPath = filepath.Join(app.dataPath, "config.ini")
	executable, _ := os.Executable()
	app.localPath = filepath.Join(filepath.Dir(executable), "data")

	app.info = log.New(os.Stdout, "[INFO] ", log.Ltime)
	app.err = log.New(os.Stdout, "[ERROR] ", log.Ltime|log.Lshortfile)

	if firstCall {
		DATA = flag.String("data", app.dataPath, "alternate path to data directory.")
		CONFIG = flag.String("config", app.configPath, "alternate path to config file.")
		HOST = flag.String("host", "", "alternate address to host web ui on.")
		PORT = flag.Int("port", 0, "alternate port to host web ui on.")
		DEBUG = flag.Bool("debug", false, "Enables debug logging and exposes pprof.")
		SWAGGER = flag.Bool("swagger", false, "Enable swagger at /swagger/index.html")

		flag.Parse()
		if *SWAGGER {
			os.Setenv("SWAGGER", "1")
		}
		if *DEBUG {
			os.Setenv("DEBUG", "1")
		}
	}

	if os.Getenv("SWAGGER") == "1" {
		*SWAGGER = true
	}
	if os.Getenv("DEBUG") == "1" {
		*DEBUG = true
	}
	// attempt to apply command line flags correctly
	if app.configPath == *CONFIG && app.dataPath != *DATA {
		app.dataPath = *DATA
		app.configPath = filepath.Join(app.dataPath, "config.ini")
	} else if app.configPath != *CONFIG && app.dataPath == *DATA {
		app.configPath = *CONFIG
	} else {
		app.configPath = *CONFIG
		app.dataPath = *DATA
	}

	// env variables are necessary because syscall.Exec for self-restarts doesn't doesn't work with arguments for some reason.

	if v := os.Getenv("JFA_CONFIGPATH"); v != "" {
		app.configPath = v
	}
	if v := os.Getenv("JFA_DATAPATH"); v != "" {
		app.dataPath = v
	}

	os.Setenv("JFA_CONFIGPATH", app.configPath)
	os.Setenv("JFA_DATAPATH", app.dataPath)

	var firstRun bool
	if _, err := os.Stat(app.dataPath); os.IsNotExist(err) {
		os.Mkdir(app.dataPath, 0700)
	}
	if _, err := os.Stat(app.configPath); os.IsNotExist(err) {
		firstRun = true
		dConfigPath := filepath.Join(app.localPath, "config-default.ini")
		var dConfig *os.File
		dConfig, err = os.Open(dConfigPath)
		if err != nil {
			app.err.Fatalf("Couldn't find default config file \"%s\"", dConfigPath)
		}
		defer dConfig.Close()
		var nConfig *os.File
		nConfig, err := os.Create(app.configPath)
		if err != nil {
			app.err.Printf("Couldn't open config file for writing: \"%s\"", app.configPath)
			app.err.Fatalf("Error: %s", err)
		}
		defer nConfig.Close()
		_, err = io.Copy(nConfig, dConfig)
		if err != nil {
			app.err.Fatalf("Couldn't copy default config. To do this manually, copy\n%s\nto\n%s", dConfigPath, app.configPath)
		}
		app.info.Printf("Copied default configuration to \"%s\"", app.configPath)
	}

	var debugMode bool
	var address string
	if app.loadConfig() != nil {
		app.err.Fatalf("Failed to load config file \"%s\"", app.configPath)
	}
	lang := app.config.Section("ui").Key("language").MustString("en-us")
	app.storage.lang.FormPath = filepath.Join(app.localPath, "lang", "form", lang+".json")
	if _, err := os.Stat(app.storage.lang.FormPath); os.IsNotExist(err) {
		app.storage.lang.FormPath = filepath.Join(app.localPath, "lang", "form", "en-us.json")
	}
	app.storage.loadLang()
	app.version = app.config.Section("jellyfin").Key("version").String()
	// read from config...
	debugMode = app.config.Section("ui").Key("debug").MustBool(false)
	// then from flag
	if *DEBUG {
		debugMode = true
	}
	if debugMode {
		app.info.Print(aurora.Magenta("\n\nWARNING: Don't use debug mode in production, as it exposes pprof on the network.\n\n"))
		app.debug = log.New(os.Stdout, "[DEBUG] ", log.Ltime|log.Lshortfile)
	} else {
		app.debug = log.New(ioutil.Discard, "", 0)
	}

	if asDaemon {
		go func() {
			socket := SOCK
			os.Remove(socket)
			listener, err := net.Listen("unix", socket)
			if err != nil {
				app.err.Fatalf("Couldn't establish socket connection at %s\n", SOCK)
			}
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			go func() {
				<-c
				os.Remove(socket)
				os.Exit(1)
			}()
			defer func() {
				listener.Close()
				os.Remove(SOCK)
			}()
			for {
				con, err := listener.Accept()
				if err != nil {
					app.err.Printf("Couldn't read message on %s: %s", socket, err)
					continue
				}
				buf := make([]byte, 512)
				nr, err := con.Read(buf)
				if err != nil {
					app.err.Printf("Couldn't read message on %s: %s", socket, err)
					continue
				}
				command := string(buf[0:nr])
				if command == "stop" {
					app.shutdown()
				}
			}
		}()
	}

	if !firstRun {
		app.host = app.config.Section("ui").Key("host").String()
		app.port = app.config.Section("ui").Key("port").MustInt(8056)

		if *HOST != app.host && *HOST != "" {
			app.host = *HOST
		}
		if *PORT != app.port && *PORT > 0 {
			app.port = *PORT
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

		app.debug.Printf("Loaded config file \"%s\"", app.configPath)

		app.debug.Println("Loading storage")

		app.storage.invite_path = app.config.Section("files").Key("invites").String()
		app.storage.loadInvites()
		app.storage.emails_path = app.config.Section("files").Key("emails").String()
		app.storage.loadEmails()
		app.storage.policy_path = app.config.Section("files").Key("user_template").String()
		app.storage.loadPolicy()
		app.storage.configuration_path = app.config.Section("files").Key("user_configuration").String()
		app.storage.loadConfiguration()
		app.storage.displayprefs_path = app.config.Section("files").Key("user_displayprefs").String()
		app.storage.loadDisplayprefs()

		app.storage.profiles_path = app.config.Section("files").Key("user_profiles").String()
		app.storage.loadProfiles()

		if !(len(app.storage.policy) == 0 && len(app.storage.configuration) == 0 && len(app.storage.displayprefs) == 0) {
			app.info.Println("Migrating user template files to new profile format")
			app.storage.migrateToProfile()
			for _, path := range [3]string{app.storage.policy_path, app.storage.configuration_path, app.storage.displayprefs_path} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					dir, fname := filepath.Split(path)
					newFname := strings.Replace(fname, ".json", ".old.json", 1)
					err := os.Rename(path, filepath.Join(dir, newFname))
					if err != nil {
						app.err.Fatalf("Failed to rename %s: %s", fname, err)
					}
				}
			}
			app.info.Println("In case of a problem, your original files have been renamed to <file>.old.json")
			app.storage.storeProfiles()
		}

		if app.config.Section("ombi").Key("enabled").MustBool(false) {
			app.storage.ombi_path = app.config.Section("files").Key("ombi_template").String()
			app.storage.loadOmbiTemplate()
			ombiServer := app.config.Section("ombi").Key("server").String()
			app.ombi = ombi.NewOmbi(
				ombiServer,
				app.config.Section("ombi").Key("api_key").String(),
				common.NewTimeoutHandler("Ombi", ombiServer, true),
			)

		}

		app.configBasePath = filepath.Join(app.localPath, "config-base.json")
		configBase, _ := ioutil.ReadFile(app.configBasePath)
		json.Unmarshal(configBase, &app.configBase)

		themes := map[string]string{
			"Jellyfin (Dark)": "dark-theme",
			"Default (Light)": "light-theme",
		}
		if app.config.Section("ui").Key("theme").String() == "Bootstrap (Light)" {
			app.config.Section("ui").Key("theme").SetValue("Default (Light)")
		}
		if val, ok := themes[app.config.Section("ui").Key("theme").String()]; ok {
			app.cssClass = val
		}
		secret, err := generateSecret(16)
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
		cacheTimeout := int(app.config.Section("jellyfin").Key("cache_timeout").MustUint(30))
		mediaBrowser := app.config.Section("jellyfin").Key("type").String()
		if mediaBrowser == "emby" {
			app.info.Println("Using Emby server type")
			app.jf, _ = emby.NewEmby(
				server,
				app.config.Section("jellyfin").Key("client").String(),
				app.config.Section("jellyfin").Key("version").String(),
				app.config.Section("jellyfin").Key("device").String(),
				app.config.Section("jellyfin").Key("device_id").String(),
				common.NewTimeoutHandler("Emby", server, true),
				cacheTimeout,
			)
		} else {
			app.info.Println("Using Jellyfin server type")
			app.jf, _ = jfapi.NewJellyfin(
				server,
				app.config.Section("jellyfin").Key("client").String(),
				app.config.Section("jellyfin").Key("version").String(),
				app.config.Section("jellyfin").Key("device").String(),
				app.config.Section("jellyfin").Key("device_id").String(),
				common.NewTimeoutHandler("Jellyfin", server, true),
				cacheTimeout,
			)
		}
		var status int
		_, status, err = app.jf.Authenticate(app.config.Section("jellyfin").Key("username").String(), app.config.Section("jellyfin").Key("password").String())
		if status != 200 || err != nil {
			app.err.Fatalf("Failed to authenticate with Jellyfin @ %s: Code %d", server, status)
		}
		app.info.Printf("Authenticated with %s", server)
		// from 10.7.0, jellyfin may hyphenate user IDs. This checks if the version is equal or higher.
		checkVersion := func(version string) int {
			numberStrings := strings.Split(version, ".")
			n := 0
			for _, s := range numberStrings {
				num, err := strconv.Atoi(s)
				if err == nil {
					n += num
				}
			}
			return n
		}
		if checkVersion(app.jf.ServerInfo.Version) >= checkVersion("10.7.0") {
			// Get users to check if server uses hyphenated userIDs
			app.jf.GetUsers(false)

			noHyphens := true
			for id := range app.storage.emails {
				if strings.Contains(id, "-") {
					noHyphens = false
					break
				}
			}
			if noHyphens == app.jf.Hyphens {
				var newEmails map[string]interface{}
				var status int
				var err error
				if app.jf.Hyphens {
					app.info.Println(aurora.Yellow("Your build of Jellyfin appears to hypenate user IDs. Your emails.json file will be modified to match."))
					time.Sleep(time.Second * time.Duration(3))
					newEmails, status, err = app.hyphenateEmailStorage(app.storage.emails)
				} else {
					app.info.Println(aurora.Yellow("Your emails.json file uses hyphens, but the Jellyfin server no longer does. It will be modified."))
					time.Sleep(time.Second * time.Duration(3))
					newEmails, status, err = app.deHyphenateEmailStorage(app.storage.emails)
				}
				if status != 200 || err != nil {
					app.err.Printf("Failed to get users from Jellyfin: Code %d", status)
					app.debug.Printf("Error: %s", err)
					app.err.Fatalf("Couldn't upgrade emails.json")
				}
				bakFile := app.storage.emails_path + ".bak"
				err = storeJSON(bakFile, app.storage.emails)
				if err != nil {
					app.err.Fatalf("couldn't store emails.json backup: %s", err)
				}
				app.storage.emails = newEmails
				err = app.storage.storeEmails()
				if err != nil {
					app.err.Fatalf("couldn't store emails.json: %s", err)
				}
			}
		}
		app.authJf, _ = jfapi.NewJellyfin(server, "jfa-go", app.version, "auth", "auth", common.NewTimeoutHandler("Jellyfin", server, true), cacheTimeout)

		app.loadStrftime()

		validatorConf := ValidatorConf{
			"length":    app.config.Section("password_validation").Key("min_length").MustInt(0),
			"uppercase": app.config.Section("password_validation").Key("upper").MustInt(0),
			"lowercase": app.config.Section("password_validation").Key("lower").MustInt(0),
			"number":    app.config.Section("password_validation").Key("number").MustInt(0),
			"special":   app.config.Section("password_validation").Key("special").MustInt(0),
		}
		if !app.config.Section("password_validation").Key("enabled").MustBool(false) {
			for key := range validatorConf {
				validatorConf[key] = 0
			}
		}
		app.validator.init(validatorConf)

		if TEST {
			test(app)
			os.Exit(0)
		}

		inviteDaemon := newRepeater(time.Duration(60*time.Second), app)
		go inviteDaemon.run()

		if app.config.Section("password_resets").Key("enabled").MustBool(false) {
			go app.StartPWR()
		}
	} else {
		debugMode = false
		address = "0.0.0.0:8056"
	}
	app.info.Println("Loading routes")
	if debugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	setGinLogger(router, debugMode)

	router.Use(gin.Recovery())
	router.Use(static.Serve("/", static.LocalFile(filepath.Join(app.localPath, "web"), false)))
	app.loadHTML(router)
	router.NoRoute(app.NoRouteHandler)
	if debugMode {
		app.debug.Println("Loading pprof")
		pprof.Register(router)
	}
	if !firstRun {
		router.GET("/", app.AdminPage)
		router.GET("/accounts", app.AdminPage)
		router.GET("/settings", app.AdminPage)

		router.GET("/token/login", app.getTokenLogin)
		router.GET("/token/refresh", app.getTokenRefresh)
		router.POST("/newUser", app.NewUser)
		router.Use(static.Serve("/invite/", static.LocalFile(filepath.Join(app.localPath, "web"), false)))
		router.GET("/invite/:invCode", app.InviteProxy)
		if *SWAGGER {
			app.info.Print(aurora.Magenta("\n\nWARNING: Swagger should not be used on a public instance.\n\n"))
			router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}
		api := router.Group("/", app.webAuth())
		router.POST("/logout", app.Logout)
		api.DELETE("/users", app.DeleteUser)
		api.GET("/users", app.GetUsers)
		api.POST("/users", app.NewUserAdmin)
		api.POST("/invites", app.GenerateInvite)
		api.GET("/invites", app.GetInvites)
		api.DELETE("/invites", app.DeleteInvite)
		api.POST("/invites/profile", app.SetProfile)
		api.GET("/profiles", app.GetProfiles)
		api.POST("/profiles/default", app.SetDefaultProfile)
		api.POST("/profiles", app.CreateProfile)
		api.DELETE("/profiles", app.DeleteProfile)
		api.POST("/invites/notify", app.SetNotify)
		api.POST("/users/emails", app.ModifyEmails)
		// api.POST("/setDefaults", app.SetDefaults)
		api.POST("/users/settings", app.ApplySettings)
		api.GET("/config", app.GetConfig)
		api.POST("/config", app.ModifyConfig)
		if app.config.Section("ombi").Key("enabled").MustBool(false) {
			api.GET("/ombi/users", app.OmbiUsers)
			api.POST("/ombi/defaults", app.SetOmbiDefaults)
		}
		app.info.Printf("Starting router @ %s", address)
	} else {
		router.GET("/", func(gc *gin.Context) {
			gc.HTML(200, "setup.html", gin.H{})
		})
		router.POST("/jellyfin/test", app.TestJF)
		router.POST("/config", app.ModifyConfig)
		app.info.Printf("Loading setup @ %s", address)
	}

	SRV = &http.Server{
		Addr:    address,
		Handler: router,
	}
	go func() {
		if err := SRV.ListenAndServe(); err != nil {
			app.err.Printf("Failure serving: %s", err)
		}
	}()
	app.quit = make(chan os.Signal)
	signal.Notify(app.quit, os.Interrupt)
	go func() {
		for range app.quit {
			app.shutdown()
		}
	}()
	for range RESTART {
		cntx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := SRV.Shutdown(cntx); err != nil {
			app.err.Fatalf("Server shutdown error: %s", err)
		}
		return
	}
}

func (app *appContext) shutdown() {
	app.info.Println("Shutting down...")

	cntx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := SRV.Shutdown(cntx); err != nil {
		app.err.Fatalf("Server shutdown error: %s", err)
	}
	os.Exit(1)
}

func flagPassed(name string) (found bool) {
	for _, f := range os.Args {
		if f == name {
			found = true
		}
	}
	return
}

// @title jfa-go internal API
// @version 0.2.0
// @description API for the jfa-go frontend
// @contact.name Harvey Tindall
// @contact.email hrfee@protonmail.ch
// @license.name MIT
// @license.url https://raw.githubusercontent.com/hrfee/jfa-go/main/LICENSE
// @BasePath /

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization

// @securityDefinitions.basic getTokenAuth
// @name getTokenAuth

// @tag.name Auth
// @tag.description --------Get a token here first!--------

// @tag.name Users
// @tag.description Jellyfin user related operations.

// @tag.name Invites
// @tag.description Invite related operations.

// @tag.name Profiles & Settings
// @tag.description Profile and settings related operations.

// @tag.name Configuration
// @tag.description jfa-go settings.

// @tag.name Ombi
// @tag.description Ombi related operations.

// @tag.name Other
// @tag.description Things that dont fit elsewhere.

func printVersion() {
	fmt.Print(aurora.Sprintf(aurora.Magenta("jfa-go version: %s (%s)\n"), aurora.BrightWhite(VERSION), aurora.White(COMMIT)))
}

func main() {
	printVersion()
	folder := "/tmp"
	if PLATFORM == "windows" {
		folder = os.Getenv("TEMP")
	}
	SOCK = filepath.Join(folder, SOCK)
	fmt.Println("Socket:", SOCK)
	if flagPassed("test") {
		TEST = true
	}
	if flagPassed("start") {
		args := []string{}
		for i, f := range os.Args {
			if f == "start" {
				args = append(args, "daemon")
			} else if i != 0 {
				args = append(args, f)
			}
		}
		cmd := exec.Command(os.Args[0], args...)
		cmd.Start()
		os.Exit(1)
	} else if flagPassed("stop") {
		con, err := net.Dial("unix", SOCK)
		if err != nil {
			fmt.Printf("Couldn't dial socket %s, are you sure jfa-go is running?\n", SOCK)
			os.Exit(1)
		}
		_, err = con.Write([]byte("stop"))
		if err != nil {
			fmt.Printf("Couldn't send command to socket %s, are you sure jfa-go is running?\n", SOCK)
			os.Exit(1)
		}
		fmt.Println("Sent.")
	} else if flagPassed("daemon") {
		start(true, true)
	} else {
		RESTART = make(chan bool, 1)
		start(false, true)
		for {
			printVersion()
			start(false, false)
		}
	}
}
