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
	"strings"
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
	invalidTokens    []string
	jf               *Jellyfin
	authJf           *Jellyfin
	ombi             *Ombi
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
)

func test(app *appContext) {
	fmt.Printf("\n\n----\n\n")
	settings := map[string]interface{}{
		"server":         app.jf.server,
		"server version": app.jf.serverInfo.Version,
		"server name":    app.jf.serverInfo.Name,
		"authenticated?": app.jf.authenticated,
		"access token":   app.jf.accessToken,
		"username":       app.jf.username,
	}
	for n, v := range settings {
		fmt.Println(n, ":", v)
	}
	users, status, err := app.jf.getUsers(false)
	fmt.Printf("getUsers: code %d err %s maplength %d\n", status, err, len(users))
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
	user, status, err := app.jf.userByName(username, false)
	fmt.Printf("userByName (%s): code %d err %s", username, status, err)
	out, err := json.MarshalIndent(user, "", "  ")
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
	app.data_path = filepath.Join(userConfigDir, "jfa-go")
	app.config_path = filepath.Join(app.data_path, "config.ini")
	executable, _ := os.Executable()
	app.local_path = filepath.Join(filepath.Dir(executable), "data")

	app.info = log.New(os.Stdout, "[INFO] ", log.Ltime)
	app.err = log.New(os.Stdout, "[ERROR] ", log.Ltime|log.Lshortfile)

	if firstCall {
		DATA = flag.String("data", app.data_path, "alternate path to data directory.")
		CONFIG = flag.String("config", app.config_path, "alternate path to config file.")
		HOST = flag.String("host", "", "alternate address to host web ui on.")
		PORT = flag.Int("port", 0, "alternate port to host web ui on.")
		DEBUG = flag.Bool("debug", false, "Enables debug logging and exposes pprof.")

		flag.Parse()
	}

	// attempt to apply command line flags correctly
	if app.config_path == *CONFIG && app.data_path != *DATA {
		app.data_path = *DATA
		app.config_path = filepath.Join(app.data_path, "config.ini")
	} else if app.config_path != *CONFIG && app.data_path == *DATA {
		app.config_path = *CONFIG
	} else {
		app.config_path = *CONFIG
		app.data_path = *DATA
	}

	// env variables are necessary because syscall.Exec for self-restarts doesn't doesn't work with arguments for some reason.

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
	// read from config...
	debugMode = app.config.Section("ui").Key("debug").MustBool(false)
	// then from flag
	if *DEBUG {
		debugMode = true
	}
	if debugMode {
		app.info.Println("WARNING: Don't use debug mode in production, as it exposes pprof on the network.")
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

		app.debug.Printf("Loaded config file \"%s\"", app.config_path)

		if app.config.Section("ui").Key("bs5").MustBool(false) {
			app.cssFile = "bs5-jf.css"
			app.bsVersion = 5
		} else {
			app.cssFile = "bs4-jf.css"
			app.bsVersion = 4
		}

		app.debug.Println("Loading storage")

		// app.storage.invite_path = filepath.Join(app.data_path, "invites.json")
		app.storage.invite_path = app.config.Section("files").Key("invites").String()
		app.storage.loadInvites()
		// app.storage.emails_path = filepath.Join(app.data_path, "emails.json")
		app.storage.emails_path = app.config.Section("files").Key("emails").String()
		app.storage.loadEmails()
		// app.storage.policy_path = filepath.Join(app.data_path, "user_template.json")
		app.storage.policy_path = app.config.Section("files").Key("user_template").String()
		app.storage.loadPolicy()
		// app.storage.configuration_path = filepath.Join(app.data_path, "user_configuration.json")
		app.storage.configuration_path = app.config.Section("files").Key("user_configuration").String()
		app.storage.loadConfiguration()
		// app.storage.displayprefs_path = filepath.Join(app.data_path, "user_displayprefs.json")
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
			app.ombi = newOmbi(
				app.config.Section("ombi").Key("server").String(),
				app.config.Section("ombi").Key("api_key").String(),
				true,
			)
		}

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
		app.jf, _ = newJellyfin(server, "jfa-go", app.version, "hrfee-arch", "hrfee-arch")
		var status int
		_, status, err = app.jf.authenticate(app.config.Section("jellyfin").Key("username").String(), app.config.Section("jellyfin").Key("password").String())
		if status != 200 || err != nil {
			app.err.Fatalf("Failed to authenticate with Jellyfin @ %s: Code %d", server, status)
		}
		app.info.Printf("Authenticated with %s", server)
		app.authJf, _ = newJellyfin(server, "jfa-go", app.version, "auth", "auth")

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

		if TEST {
			test(app)
			os.Exit(0)
		}

		inviteDaemon := NewRepeater(time.Duration(60*time.Second), app)
		go inviteDaemon.Run()

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
	router.Use(static.Serve("/", static.LocalFile(filepath.Join(app.local_path, "static"), false)))
	router.LoadHTMLGlob(filepath.Join(app.local_path, "templates", "*"))
	router.NoRoute(app.NoRouteHandler)
	if debugMode {
		app.debug.Println("Loading pprof")
		pprof.Register(router)
	}
	if !firstRun {
		router.GET("/", app.AdminPage)
		router.GET("/getToken", app.getToken)
		router.POST("/newUser", app.NewUser)
		router.Use(static.Serve("/invite/", static.LocalFile(filepath.Join(app.local_path, "static"), false)))
		router.GET("/invite/:invCode", app.InviteProxy)
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

func main() {
	fmt.Printf("jfa-go version: %s (%s)\n", VERSION, COMMIT)
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
			fmt.Printf("jfa-go version: %s (%s)\n", VERSION, COMMIT)
			start(false, false)
		}
	}
}
