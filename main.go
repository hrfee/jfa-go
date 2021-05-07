package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
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

	"github.com/fatih/color"
	"github.com/hrfee/jfa-go/common"
	_ "github.com/hrfee/jfa-go/docs"
	"github.com/hrfee/jfa-go/logger"
	"github.com/hrfee/jfa-go/ombi"
	"github.com/hrfee/mediabrowser"
	"github.com/lithammer/shortuuid/v3"
	"gopkg.in/ini.v1"
)

var (
	PLATFORM           string = runtime.GOOS
	SOCK               string = "jfa-go.sock"
	SRV                *http.Server
	RESTART            chan bool
	DATA, CONFIG, HOST *string
	PORT               *int
	DEBUG              *bool
	PPROF              *bool
	TEST               bool
	SWAGGER            *bool
	warning            = color.New(color.FgYellow).SprintfFunc()
	info               = color.New(color.FgMagenta).SprintfFunc()
	hiwhite            = color.New(color.FgHiWhite).SprintfFunc()
	white              = color.New(color.FgWhite).SprintfFunc()
	version            string
	commit             string
)

var temp = func() string {
	temp := "/tmp"
	if PLATFORM == "windows" {
		temp = os.Getenv("TEMP")
	}
	return temp
}()

var serverTypes = map[string]string{
	"jellyfin": "Jellyfin",
	"emby":     "Emby (experimental)",
}
var serverType = mediabrowser.JellyfinServer
var substituteStrings = ""

// User is used for auth purposes.
type User struct {
	UserID   string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// contains (almost) everything the application needs, essentially. This was a dumb design decision imo.
type appContext struct {
	// defaults         *Config
	config         *ini.File
	configPath     string
	configBasePath string
	configBase     settings
	dataPath       string
	webFS          httpFS
	cssClass       string // Default theme, "light-theme"|"dark-theme".
	jellyfinLogin  bool
	users          []User
	invalidTokens  []string
	// Keeping jf name because I can't think of a better one
	jf               *mediabrowser.MediaBrowser
	authJf           *mediabrowser.MediaBrowser
	ombi             *ombi.Ombi
	datePattern      string
	timePattern      string
	storage          Storage
	validator        Validator
	email            *Emailer
	telegram         *TelegramDaemon
	info, debug, err logger.Logger
	host             string
	port             int
	version          string
	quit             chan os.Signal
	URLBase          string
	updater          *Updater
	newUpdate        bool // Whether whatever's in update is new.
	tag              Tag
	update           Update
}

func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), err
}

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
		set default config and data paths
		data: Contains invites.json, emails.json, user_profile.json, etc.
		config: config.ini. Usually in data, but can be changed via -config.
		localFS: jfa-go's internal data. On internal builds, this is contained within the binary.
				 On external builds, the directory is named "data" and placed next to the executable.
	*/
	userConfigDir, _ := os.UserConfigDir()
	app.dataPath = filepath.Join(userConfigDir, "jfa-go")
	app.configPath = filepath.Join(app.dataPath, "config.ini")
	// gin-static doesn't just take a plain http.FileSystem, so we implement it's ServeFileSystem. See static.go.
	app.webFS = httpFS{
		hfs: http.FS(localFS),
		fs:  localFS,
	}

	app.info = logger.NewLogger(os.Stdout, "[INFO] ", log.Ltime, color.FgHiWhite)
	app.err = logger.NewLogger(os.Stdout, "[ERROR] ", log.Ltime|log.Lshortfile, color.FgRed)

	app.loadArgs(firstCall)

	var firstRun bool
	if _, err := os.Stat(app.dataPath); os.IsNotExist(err) {
		os.Mkdir(app.dataPath, 0700)
	}
	if _, err := os.Stat(app.configPath); os.IsNotExist(err) {
		firstRun = true
		dConfig, err := fs.ReadFile(localFS, "config-default.ini")
		if err != nil {
			app.err.Fatalf("Couldn't find default config file")
		}
		nConfig, err := os.Create(app.configPath)
		if err != nil {
			app.err.Printf("Couldn't open config file for writing: \"%s\"", app.configPath)
			app.err.Fatalf("Error: %s", err)
		}
		defer nConfig.Close()
		_, err = nConfig.Write(dConfig)
		if err != nil {
			app.err.Fatalf("Couldn't copy default config.")
		}
		app.info.Printf("Copied default configuration to \"%s\"", app.configPath)
	}

	var debugMode bool
	var address string
	if err := app.loadConfig(); err != nil {
		app.err.Fatalf("Failed to load config file \"%s\": %v", app.configPath, err)
	}
	app.version = app.config.Section("jellyfin").Key("version").String()
	// read from config...
	debugMode = app.config.Section("ui").Key("debug").MustBool(false)
	// then from flag
	if *DEBUG {
		debugMode = true
	}
	if debugMode {
		app.debug = logger.NewLogger(os.Stdout, "[DEBUG] ", log.Ltime|log.Lshortfile, color.FgYellow)
	} else {
		app.debug = logger.EmptyLogger(false)
	}
	if *PPROF {
		app.info.Print(warning("\n\nWARNING: Don't use pprof in production.\n\n"))
	}

	// Starts listener to receive commands over a unix socket. Use with 'jfa-go start/stop'
	if asDaemon {
		go func() {
			os.Remove(SOCK)
			listener, err := net.Listen("unix", SOCK)
			if err != nil {
				app.err.Fatalf("Couldn't establish socket connection at %s\n", SOCK)
			}
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			go func() {
				<-c
				os.Remove(SOCK)
				os.Exit(1)
			}()
			defer func() {
				listener.Close()
				os.Remove(SOCK)
			}()
			for {
				con, err := listener.Accept()
				if err != nil {
					app.err.Printf("Couldn't read message on %s: %s", SOCK, err)
					continue
				}
				buf := make([]byte, 512)
				nr, err := con.Read(buf)
				if err != nil {
					app.err.Printf("Couldn't read message on %s: %s", SOCK, err)
					continue
				}
				command := string(buf[0:nr])
				if command == "stop" {
					app.shutdown()
				}
			}
		}()
	}

	app.storage.lang.CommonPath = "common"
	app.storage.lang.FormPath = "form"
	app.storage.lang.AdminPath = "admin"
	app.storage.lang.EmailPath = "email"
	app.storage.lang.TelegramPath = "telegram"
	app.storage.lang.PasswordResetPath = "pwreset"
	externalLang := app.config.Section("files").Key("lang_files").MustString("")
	var err error
	if externalLang == "" {
		err = app.storage.loadLang(langFS)
	} else {
		err = app.storage.loadLang(langFS, os.DirFS(externalLang))
	}
	if err != nil {
		app.info.Fatalf("Failed to load language files: %+v\n", err)
	}

	if !firstRun {
		app.host = app.config.Section("ui").Key("host").String()
		if app.config.Section("advanced").Key("tls").MustBool(false) {
			app.info.Println("Using TLS/HTTP2")
			app.port = app.config.Section("advanced").Key("tls_port").MustInt(8057)
		} else {
			app.port = app.config.Section("ui").Key("port").MustInt(8056)
		}

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
		if err := app.storage.loadInvites(); err != nil {
			app.err.Printf("Failed to load Invites: %v", err)
		}
		app.storage.emails_path = app.config.Section("files").Key("emails").String()
		if err := app.storage.loadEmails(); err != nil {
			app.err.Printf("Failed to load Emails: %v", err)
		}
		app.storage.policy_path = app.config.Section("files").Key("user_template").String()
		if err := app.storage.loadPolicy(); err != nil {
			app.err.Printf("Failed to load Policy: %v", err)
		}
		app.storage.configuration_path = app.config.Section("files").Key("user_configuration").String()
		if err := app.storage.loadConfiguration(); err != nil {
			app.err.Printf("Failed to load Configuration: %v", err)
		}
		app.storage.displayprefs_path = app.config.Section("files").Key("user_displayprefs").String()
		if err := app.storage.loadDisplayprefs(); err != nil {
			app.err.Printf("Failed to load Displayprefs: %v", err)
		}
		app.storage.users_path = app.config.Section("files").Key("users").String()
		if err := app.storage.loadUsers(); err != nil {
			app.err.Printf("Failed to load Users: %v", err)
		}
		app.storage.telegram_path = app.config.Section("files").Key("telegram_users").String()
		if err := app.storage.loadTelegramUsers(); err != nil {
			app.err.Printf("Failed to load Telegram users: %v", err)
		}

		app.storage.profiles_path = app.config.Section("files").Key("user_profiles").String()
		app.storage.loadProfiles()
		// Migrate from pre-0.2.0 user templates to profiles
		if !(app.storage.policy.BlockedTags == nil && app.storage.configuration.GroupedFolders == nil && len(app.storage.displayprefs) == 0) {
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

		// Read config-base for settings on web.
		app.configBasePath = "config-base.json"
		configBase, _ := fs.ReadFile(localFS, app.configBasePath)
		json.Unmarshal(configBase, &app.configBase)

		themes := map[string]string{
			"Jellyfin (Dark)": "dark-theme",
			"Default (Light)": "light-theme",
		}
		// For move from Bootstrap to a17t (0.2.5)
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

		// Initialize jellyfin/emby connection
		server := app.config.Section("jellyfin").Key("server").String()
		cacheTimeout := int(app.config.Section("jellyfin").Key("cache_timeout").MustUint(30))
		stringServerType := app.config.Section("jellyfin").Key("type").String()
		timeoutHandler := mediabrowser.NewNamedTimeoutHandler("Jellyfin", server, true)
		if stringServerType == "emby" {
			serverType = mediabrowser.EmbyServer
			timeoutHandler = mediabrowser.NewNamedTimeoutHandler("Emby", server, true)
			app.info.Println("Using Emby server type")
			fmt.Println(warning("WARNING: Emby compatibility is experimental, and support is limited.\nPassword resets are not available."))
		} else {
			app.info.Println("Using Jellyfin server type")
		}

		app.jf, _ = mediabrowser.NewServer(
			serverType,
			server,
			app.config.Section("jellyfin").Key("client").String(),
			app.config.Section("jellyfin").Key("version").String(),
			app.config.Section("jellyfin").Key("device").String(),
			app.config.Section("jellyfin").Key("device_id").String(),
			timeoutHandler,
			cacheTimeout,
		)
		if debugMode {
			app.jf.Verbose = true
		}
		var status int
		_, status, err = app.jf.Authenticate(app.config.Section("jellyfin").Key("username").String(), app.config.Section("jellyfin").Key("password").String())
		if status != 200 || err != nil {
			app.err.Fatalf("Failed to authenticate with Jellyfin @ %s (%d): %v", server, status, err)
		}
		app.info.Printf("Authenticated with %s", server)
		/* A couple of unstable Jellyfin 10.7.0 releases decided to hyphenate user IDs.
		This checks if the version is equal or higher. */
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
		if serverType == mediabrowser.JellyfinServer && checkVersion(app.jf.ServerInfo.Version) >= checkVersion("10.7.0") {
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
				var newUsers map[string]time.Time
				var status, status2 int
				var err, err2 error
				if app.jf.Hyphens {
					app.info.Println(info("Your build of Jellyfin appears to hypenate user IDs. Your emails.json/users.json file will be modified to match."))
					time.Sleep(time.Second * time.Duration(3))
					newEmails, status, err = app.hyphenateEmailStorage(app.storage.emails)
					newUsers, status2, err2 = app.hyphenateUserStorage(app.storage.users)
				} else {
					app.info.Println(info("Your emails.json/users.json file uses hyphens, but the Jellyfin server no longer does. It will be modified."))
					time.Sleep(time.Second * time.Duration(3))
					newEmails, status, err = app.deHyphenateEmailStorage(app.storage.emails)
					newUsers, status2, err2 = app.deHyphenateUserStorage(app.storage.users)
				}
				if status != 200 || err != nil {
					app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
					app.err.Fatalf("Couldn't upgrade emails.json")
				}
				if status2 != 200 || err2 != nil {
					app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
					app.err.Fatalf("Couldn't upgrade users.json")
				}
				emailBakFile := app.storage.emails_path + ".bak"
				usersBakFile := app.storage.users_path + ".bak"
				err = storeJSON(emailBakFile, app.storage.emails)
				err2 = storeJSON(usersBakFile, app.storage.users)
				if err != nil {
					app.err.Fatalf("couldn't store emails.json backup: %v", err)
				}
				if err2 != nil {
					app.err.Fatalf("couldn't store users.json backup: %v", err)
				}
				app.storage.emails = newEmails
				app.storage.users = newUsers
				err = app.storage.storeEmails()
				err2 = app.storage.storeUsers()
				if err != nil {
					app.err.Fatalf("couldn't store emails.json: %v", err)
				}
				if err2 != nil {
					app.err.Fatalf("couldn't store users.json: %v", err)
				}
			}
		}

		// Auth (manual user/pass or jellyfin)
		app.jellyfinLogin = true
		if jfLogin, _ := app.config.Section("ui").Key("jellyfin_login").Bool(); !jfLogin {
			app.jellyfinLogin = false
			user := User{}
			user.UserID = shortuuid.New()
			user.Username = app.config.Section("ui").Key("username").String()
			user.Password = app.config.Section("ui").Key("password").String()
			app.users = append(app.users, user)
		} else {
			app.debug.Println("Using Jellyfin for authentication")
			app.authJf, _ = mediabrowser.NewServer(serverType, server, "jfa-go", app.version, "auth", "auth", timeoutHandler, cacheTimeout)
			if debugMode {
				app.authJf.Verbose = true
			}
		}

		// Since email depends on language, the email reload in loadConfig won't work first time.
		app.email = NewEmailer(app)
		app.loadStrftime()

		var validatorConf ValidatorConf

		if !app.config.Section("password_validation").Key("enabled").MustBool(false) {
			validatorConf = ValidatorConf{}
		} else {
			validatorConf = ValidatorConf{
				"length":    app.config.Section("password_validation").Key("min_length").MustInt(0),
				"uppercase": app.config.Section("password_validation").Key("upper").MustInt(0),
				"lowercase": app.config.Section("password_validation").Key("lower").MustInt(0),
				"number":    app.config.Section("password_validation").Key("number").MustInt(0),
				"special":   app.config.Section("password_validation").Key("special").MustInt(0),
			}
		}
		app.validator.init(validatorConf)

		// Test mode for testing connection to Jellyfin, accessed with 'jfa-go test'
		if TEST {
			test(app)
			os.Exit(0)
		}

		invDaemon := newInviteDaemon(time.Duration(60*time.Second), app)
		go invDaemon.run()
		defer invDaemon.shutdown()

		userDaemon := newUserDaemon(time.Duration(60*time.Second), app)
		go userDaemon.run()
		defer userDaemon.shutdown()

		if app.config.Section("password_resets").Key("enabled").MustBool(false) && serverType == mediabrowser.JellyfinServer {
			go app.StartPWR()
		}

		if app.config.Section("updates").Key("enabled").MustBool(false) {
			go app.checkForUpdates()
		}

		if app.config.Section("telegram").Key("enabled").MustBool(false) {
			app.telegram, err = newTelegramDaemon(app)
			if err != nil {
				app.err.Printf("Failed to authenticate with Telegram: %v", err)
			} else {
				go app.telegram.run()
				defer app.telegram.Shutdown()
			}
		}
	} else {
		debugMode = false
		address = "0.0.0.0:8056"
		app.storage.lang.SetupPath = "setup"
		err := app.storage.loadLangSetup(langFS)
		if err != nil {
			app.info.Fatalf("Failed to load language files: %+v\n", err)
		}
	}

	cssHeader = app.loadCSSHeader()
	// workaround for potentially broken windows mime types
	mime.AddExtensionType(".js", "application/javascript")

	app.info.Println("Initializing router")
	router := app.loadRouter(address, debugMode)
	app.info.Println("Loading routes")
	if !firstRun {
		app.loadRoutes(router)
	} else {
		app.loadSetup(router)
		app.info.Printf("Loading setup @ %s", address)
	}
	go func() {
		if app.config.Section("advanced").Key("tls").MustBool(false) {
			cert := app.config.Section("advanced").Key("tls_cert").MustString("")
			key := app.config.Section("advanced").Key("tls_key").MustString("")
			if err := SRV.ListenAndServeTLS(cert, key); err != nil {
				app.err.Printf("Failure serving: %s", err)
			}
		} else {
			if err := SRV.ListenAndServe(); err != nil {
				app.err.Printf("Failure serving: %s", err)
			}
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := SRV.Shutdown(ctx); err != nil {
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
	fmt.Println(info("jfa-go version: %s (%s)\n", hiwhite(version), white(commit)))
}

func main() {
	printVersion()
	SOCK = filepath.Join(temp, SOCK)
	fmt.Println("Socket:", SOCK)
	if flagPassed("test") {
		TEST = true
	}
	loadFilesystems()
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
	} else if flagPassed("systemd") {
		service, err := fs.ReadFile(localFS, "jfa-go.service")
		if err != nil {
			fmt.Printf("Couldn't read jfa-go.service: %v\n", err)
			os.Exit(1)
		}
		absPath, err := filepath.Abs(os.Args[0])
		if err != nil {
			absPath = os.Args[0]
		}
		command := absPath
		for i, v := range os.Args {
			if i != 0 && v != "systemd" {
				command += " " + v
			}
		}
		service = []byte(strings.Replace(string(service), "{executable}", command, 1))
		err = os.WriteFile("jfa-go.service", service, 0666)
		if err != nil {
			fmt.Printf("Couldn't write jfa-go.service: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(info(`If you want to execute jfa-go with special arguments, re-run this command with them.
Move the newly created "jfa-go.service" file to ~/.config/systemd/user (Creating it if necessary).
Then run "systemctl --user daemon-reload".
You can then run:

`))
		color.New(color.FgGreen).PrintFunc()("To start: ")
		fmt.Print(info("systemctl --user start jfa-go\n\n"))
		color.New(color.FgRed).PrintFunc()("To stop: ")
		fmt.Print(info("systemctl --user stop jfa-go\n\n"))
		color.New(color.FgYellow).PrintFunc()("To restart: ")
		fmt.Print(info("systemctl --user stop jfa-go\n"))
	} else {
		RESTART = make(chan bool, 1)
		start(false, true)
		for {
			printVersion()
			start(false, false)
		}
	}
}
