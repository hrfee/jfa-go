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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/hrfee/jfa-go/common"
	_ "github.com/hrfee/jfa-go/docs"
	"github.com/hrfee/jfa-go/jellyseerr"
	"github.com/hrfee/jfa-go/logger"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/hrfee/jfa-go/ombi"
	"github.com/hrfee/mediabrowser"
	"github.com/lithammer/shortuuid/v3"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

var (
	PLATFORM           string = runtime.GOOS
	SOCK               string = "jfa-go.sock"
	SRV                *http.Server
	RESTART            chan bool
	TRAYRESTART        chan bool
	DATA, CONFIG, HOST *string
	PORT               *int
	DEBUG              *bool
	PPROF              *bool
	TEST               bool
	SWAGGER            *bool
	QUIT               = false
	RUNNING            = false
	LOGIP              = false // Log admin IPs
	LOGIPU             = false // Log user IPs
	// Used to know how many times to re-broadcast restart signal.
	RESTARTLISTENERCOUNT = 0
	warning              = color.New(color.FgYellow).SprintfFunc()
	info                 = color.New(color.FgMagenta).SprintfFunc()
	hiwhite              = color.New(color.FgHiWhite).SprintfFunc()
	white                = color.New(color.FgWhite).SprintfFunc()
	version              string
	commit               string
	buildTimeUnix        string
	builtBy              string
	buildTags            []string
	_LOADBAK             *string
	LOADBAK              = ""
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

var externalURI, externalDomain string // The latter lower-case as should be accessed through app.ExternalDomain()
var UseProxyHost bool

var datePattern, timePattern string

// User is used for auth purposes.
type User struct {
	UserID   string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Set of the usual log channels, for ease of passing between things.
type LoggerSet struct {
	info, debug, err *logger.Logger
}

// contains (almost) everything the application needs, essentially. This was a dumb design decision imo.
type appContext struct {
	// defaults         *Config
	config         *Config
	configPath     string
	configBasePath string
	configBase     common.Config
	patchedConfig  common.Config
	dataPath       string
	webFS          httpFS
	cssClass       string // Default theme, "light"|"dark".
	jellyfinLogin  bool
	adminUsers     []User
	invalidTokens  []string
	// Keeping jf name because I can't think of a better one
	jf                             *mediabrowser.MediaBrowser
	authJf                         *mediabrowser.MediaBrowser
	ombi                           *OmbiWrapper
	js                             *JellyseerrWrapper
	thirdPartyServices             []ThirdPartyService
	storage                        *Storage
	validator                      Validator
	email                          *Emailer
	telegram                       *TelegramDaemon
	discord                        *DiscordDaemon
	matrix                         *MatrixDaemon
	housekeepingDaemon, userDaemon *GenericDaemon
	contactMethods                 []ContactMethodLinker
	LoggerSet
	host                 string
	port                 int
	version              string
	updater              *Updater
	webhooks             *WebhookSender
	newUpdate            bool // Whether whatever's in update is new.
	tag                  Tag
	update               Update
	internalPWRs         map[string]InternalPWR
	pwrCaptchas          map[string]Captcha
	ConfirmationKeys     map[string]map[string]ConfirmationKey // Map of invite code to jwt to request
	confirmationKeysLock sync.Mutex
	userCache            *UserCache
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
	settings := map[string]any{
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
	users, err := app.jf.GetUsers(false)
	fmt.Printf("GetUsers: err %s maplength %d\n", err, len(users))
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
	user, err := app.jf.UserByName(username, false)
	fmt.Printf("UserByName (%s): err %v", username, err)
	out, _ := json.MarshalIndent(user, "", "  ")
	fmt.Print(string(out))
}

func start(asDaemon, firstCall bool) {
	RESTARTLISTENERCOUNT = 0
	RUNNING = true
	defer func() { RUNNING = false }()

	defer func() {
		if r := recover(); r != nil {
			Exit(r)
		}
	}()
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
	app.info.SetFatalFunc(Exit)
	app.err = logger.NewLogger(os.Stdout, "[ERROR] ", log.Ltime|log.Lshortfile, color.FgRed)
	app.err.SetFatalFunc(Exit)

	app.loadArgs(firstCall)

	var firstRun bool
	if _, err := os.Stat(app.dataPath); os.IsNotExist(err) {
		os.Mkdir(app.dataPath, 0700)
	}
	if _, err := os.Stat(app.configPath); os.IsNotExist(err) {
		firstRun = true
		dConfig, err := fs.ReadFile(localFS, "config-default.ini")
		if err != nil {
			app.err.Fatalf(lm.NoConfig)
		}
		nConfig, err := os.Create(app.configPath)
		if err != nil && os.IsNotExist(err) {
			err = os.MkdirAll(filepath.Dir(app.configPath), 0760)
		}
		if err != nil {
			app.err.Fatalf(lm.FailedWriting, app.configPath, err)
		}
		defer nConfig.Close()
		_, err = nConfig.Write(dConfig)
		if err != nil {
			app.err.Fatalf(lm.FailedCopyConfig, app.configPath, err)
		}
		app.info.Printf(lm.CopyConfig, app.configPath)
		tempConfig, _ := ini.ShadowLoad(app.configPath)
		tempConfig.Section("").Key("first_run").SetValue("true")
		tempConfig.SaveTo(app.configPath)
	}

	var debugMode bool
	var address string
	var err error = nil
	app.config, err = NewConfig(app.configPath, app.dataPath, app.LoggerSet)
	if err != nil {
		app.err.Fatalf(lm.FailedLoadConfig, app.configPath, err)
	}
	app.info.Printf(lm.LoadConfig, app.configPath)

	if app.config.Section("").Key("first_run").MustBool(false) {
		firstRun = true
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
		app.debug = logger.NewEmptyLogger()
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
				app.err.Fatalf(lm.FailedSocketConnect, SOCK, err)
			}
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
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
					app.err.Printf(lm.FailedSocketRead, SOCK, err)
					continue
				}
				buf := make([]byte, 512)
				nr, err := con.Read(buf)
				if err != nil {
					app.err.Printf(lm.FailedSocketRead, SOCK, err)
					continue
				}
				command := string(buf[0:nr])
				if command == "stop" {
					app.shutdown()
				}
			}
		}()
	}

	dbPath := filepath.Join(app.dataPath, "db")
	if debugMode {
		app.storage = NewStorage(dbPath, app.debug, generateLogActions(app.config))
	} else {
		app.storage = NewStorage(dbPath, app.debug, nil)
	}

	// Placed here, since storage.chosenXLang is set by this function.
	app.config.ReloadDependents(app)

	externalLang := app.config.Section("files").Key("lang_files").MustString("")
	if externalLang == "" {
		err = app.storage.loadLang(langFS)
	} else {
		err = app.storage.loadLang(langFS, os.DirFS(externalLang))
	}
	if err != nil {
		app.info.Fatalf(lm.FailedLangLoad, err)
	}

	// Read config-base for settings on web.
	app.configBasePath = "config-base.yaml"
	configBase, _ := fs.ReadFile(localFS, app.configBasePath)
	yaml.Unmarshal(configBase, &app.configBase)

	if !firstRun {
		app.host = app.config.Section("ui").Key("host").String()
		if app.config.Section("advanced").Key("tls").MustBool(false) {
			app.info.Println(lm.UsingTLS)
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

		// NOTE: As of writing this, the order in app.thirdPartyServices doesn't matter,
		// but in future it might (like app.contactMethods does), so append to the end!
		if app.config.Section("ombi").Key("enabled").MustBool(false) {
			app.ombi = &OmbiWrapper{}
			app.debug.Printf(lm.UsingOmbi)
			ombiServer := app.config.Section("ombi").Key("server").String()
			app.ombi.Ombi = ombi.NewOmbi(
				ombiServer,
				app.config.Section("ombi").Key("api_key").String(),
				common.NewTimeoutHandler("Ombi", ombiServer, true),
			)
			app.thirdPartyServices = append(app.thirdPartyServices, app.ombi)
		}

		if app.config.Section("jellyseerr").Key("enabled").MustBool(false) {
			app.js = &JellyseerrWrapper{}
			app.debug.Printf(lm.UsingJellyseerr)
			jellyseerrServer := app.config.Section("jellyseerr").Key("server").String()
			app.js.Jellyseerr = jellyseerr.NewJellyseerr(
				jellyseerrServer,
				app.config.Section("jellyseerr").Key("api_key").String(),
				common.NewTimeoutHandler("Jellyseerr", jellyseerrServer, true),
			)
			app.js.AutoImportUsers = app.config.Section("jellyseerr").Key("import_existing").MustBool(false)
			// app.js.LogRequestBodies = true
			app.thirdPartyServices = append(app.thirdPartyServices, app.js)

		}

		app.loadPendingBackup()
		if err := app.storage.Connect(app.config); err != nil {
			app.err.Fatalf(lm.FailedConnectDB, dbPath, err)
		}
		app.info.Printf(lm.ConnectDB, dbPath)
		defer app.storage.Close()

		// copy it to app.patchedConfig, and patch in settings from app.config, and language stuff.
		app.PatchConfigBase()

		secret, err := generateSecret(16)
		if err != nil {
			app.err.Fatal(err)
		}
		os.Setenv("JFA_SECRET", secret)

		// Initialize jellyfin/emby connection
		server := app.config.Section("jellyfin").Key("server").String()
		cacheTimeout := app.config.Section("jellyfin").Key("cache_timeout").MustInt()
		stringServerType := app.config.Section("jellyfin").Key("type").String()
		timeoutHandler := mediabrowser.NewNamedTimeoutHandler("Jellyfin", "\""+server+"\"", true)
		if stringServerType == "emby" {
			serverType = mediabrowser.EmbyServer
			timeoutHandler = mediabrowser.NewNamedTimeoutHandler("Emby", "\""+server+"\"", true)
			app.info.Println(lm.UsingEmby)
		} else {
			app.info.Println(lm.UsingJellyfin)
		}

		app.jf, err = mediabrowser.NewServer(
			serverType,
			server,
			app.config.Section("jellyfin").Key("client").String(),
			app.config.Section("jellyfin").Key("version").String(),
			app.config.Section("jellyfin").Key("device").String(),
			app.config.Section("jellyfin").Key("device_id").String(),
			timeoutHandler,
			cacheTimeout,
		)
		if err != nil {
			app.err.Fatalf(lm.FailedAuthJellyfin, server, -1, err)
		}
		/*if debugMode {
			app.jf.Verbose = true
		}*/
		// It's probably best we leave this on
		app.jf.Verbose = true

		var status int
		retryOpts := mediabrowser.MustAuthenticateOptions{
			RetryCount:  app.config.Section("advanced").Key("auth_retry_count").MustInt(6),
			RetryGap:    time.Duration(app.config.Section("advanced").Key("auth_retry_gap").MustInt(10)) * time.Second,
			LogFailures: true,
		}
		_, err = app.jf.MustAuthenticate(app.config.Section("jellyfin").Key("username").String(), app.config.Section("jellyfin").Key("password").String(), retryOpts)
		if err != nil {
			app.err.Fatalf(lm.FailedAuthJellyfin, server, status, err)
		}
		app.info.Printf(lm.AuthJellyfin, server)
		app.debug.Printf(lm.AsUser, app.jf.Username)

		runMigrations(app)

		// Auth (manual user/pass or jellyfin)
		app.jellyfinLogin = true
		if jfLogin, _ := app.config.Section("ui").Key("jellyfin_login").Bool(); !jfLogin {
			app.jellyfinLogin = false
			user := User{}
			user.UserID = shortuuid.New()
			user.Username = app.config.Section("ui").Key("username").String()
			user.Password = app.config.Section("ui").Key("password").String()
			app.adminUsers = append(app.adminUsers, user)
			app.info.Println(lm.UsingLocalAuth)
		} else {
			app.debug.Println(lm.UsingJellyfinAuth)
			app.authJf, _ = mediabrowser.NewServer(serverType, server, "jfa-go", app.version, "auth", "auth", timeoutHandler, cacheTimeout)
			if debugMode {
				app.authJf.Verbose = true
			}
		}

		app.userCache = NewUserCache(
			time.Minute*time.Duration(app.config.Section("jellyfin").Key("web_cache_async_timeout").MustInt()),
			time.Minute*time.Duration(app.config.Section("jellyfin").Key("web_cache_sync_timeout").MustInt()),
		)

		// Since email depends on language, the email reload in NewConfig won't work first time.
		// Email also handles its own proxying, as (SMTP atleast) doesn't use a HTTP transport.
		app.email = NewEmailer(app.config, app.storage, app.LoggerSet)

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

		app.housekeepingDaemon = newHousekeepingDaemon(time.Duration(60*time.Second), app)
		go app.housekeepingDaemon.run()
		defer app.housekeepingDaemon.Shutdown()

		app.userDaemon = newUserDaemon(time.Duration(60*time.Second), app)
		go app.userDaemon.run()
		defer app.userDaemon.Shutdown()

		var jellyseerrDaemon *GenericDaemon
		if app.config.Section("jellyseerr").Key("enabled").MustBool(false) && app.config.Section("jellyseerr").Key("import_existing").MustBool(false) {
			// jellyseerrDaemon = newJellyseerrDaemon(time.Duration(30*time.Second), app)
			jellyseerrDaemon = newJellyseerrDaemon(time.Duration(10*time.Minute), app)
			go jellyseerrDaemon.run()
			defer jellyseerrDaemon.Shutdown()
		}

		if app.config.Section("password_resets").Key("enabled").MustBool(false) && serverType == mediabrowser.JellyfinServer {
			go app.StartPWR()
		}

		if app.config.Section("updates").Key("enabled").MustBool(false) {
			go app.checkForUpdates()
		}

		var backupDaemon *GenericDaemon
		if app.config.Section("backups").Key("enabled").MustBool(false) {
			backupDaemon = newBackupDaemon(app)
			go backupDaemon.run()
			defer backupDaemon.Shutdown()
		}

		// NOTE: The order in which these are placed in app.contactMethods matters.
		// Add new ones to the end.
		// Proxies are added a little later through ContactMethodLinker[].SetTransport.
		if discordEnabled {
			app.discord, err = newDiscordDaemon(app)
			if err != nil {
				app.err.Printf(lm.FailedInitDiscord, err)
				discordEnabled = false
			} else {
				app.debug.Println(lm.InitDiscord)
				go app.discord.Run()
				defer app.discord.Shutdown()
				app.contactMethods = append(app.contactMethods, app.discord)
			}
		}
		if telegramEnabled {
			app.telegram, err = newTelegramDaemon(app)
			if err != nil {
				app.err.Printf(lm.FailedInitTelegram, err)
				telegramEnabled = false
			} else {
				app.debug.Println(lm.InitTelegram)
				go app.telegram.run()
				defer app.telegram.Shutdown()
				app.contactMethods = append(app.contactMethods, app.telegram)
			}
		}
		if matrixEnabled {
			app.matrix, err = newMatrixDaemon(app)
			if err != nil {
				app.err.Printf(lm.FailedInitMatrix, err)
				matrixEnabled = false
			} else {
				app.debug.Println(lm.InitMatrix)
				go app.matrix.run()
				defer app.matrix.Shutdown()
				app.contactMethods = append(app.contactMethods, app.matrix)
			}
		}

		// Non-consequential if we don't need it
		app.webhooks = NewWebhookSender(
			common.NewTimeoutHandler("Webhook", "?", true),
			app.debug,
		)

		// Updater proxy set in config.go, don't worry!
		if app.config.proxyConfig != nil {
			app.jf.SetTransport(app.config.proxyTransport)
			for _, c := range app.thirdPartyServices {
				c.SetTransport(app.config.proxyTransport)
			}
			for _, c := range app.contactMethods {
				c.SetTransport(app.config.proxyTransport)
			}
		}
	} else {
		debugMode = false
		if *PORT != app.port && *PORT > 0 {
			app.port = *PORT
		} else {
			app.port = 8056
		}
		if *HOST != app.host && *HOST != "" {
			app.host = *HOST
		} else {
			app.host = "0.0.0.0"
		}
		address = fmt.Sprintf("%s:%d", app.host, app.port)
		err := app.storage.loadLangSetup(langFS)
		if err != nil {
			app.info.Fatalf(lm.FailedLangLoad, err)
		}
	}

	cssHeader = app.loadCSSHeader()
	// workaround for potentially broken windows mime types
	mime.AddExtensionType(".js", "application/javascript")

	app.info.Println(lm.InitRouter)
	router := app.loadRouter(address, debugMode)
	app.info.Println(lm.LoadRoutes)
	if !firstRun {
		app.loadRoutes(router)
	} else {
		app.loadSetup(router)
		app.info.Printf(lm.LoadingSetup, address)
	}
	go func() {
		if app.config.Section("advanced").Key("tls").MustBool(false) {
			cert := app.config.Section("advanced").Key("tls_cert").MustString("")
			key := app.config.Section("advanced").Key("tls_key").MustString("")
			if err := SRV.ListenAndServeTLS(cert, key); err != nil {
				filesToCheck := []string{cert, key}
				fileNames := []string{lm.InvalidSSLCert, lm.InvalidSSLKey}
				for i, v := range filesToCheck {
					_, err := os.Stat(v)
					if err != nil {
						app.err.Printf(fileNames[i], v, err)
					}
				}

				if err == http.ErrServerClosed {
					app.err.Printf(lm.FailServeSSL, err)
				} else {
					app.err.Fatalf(lm.FailServeSSL, err)
				}
			}
		} else {
			if err := SRV.ListenAndServe(); err != nil {
				if err == http.ErrServerClosed {
					app.err.Printf(lm.FailServe, err)
				} else {
					app.err.Fatalf(lm.FailServe, err)
				}
			}
		}
	}()
	if firstRun {
		app.info.Printf(lm.ServingSetup, address)
	} else {
		app.info.Printf(lm.Serving, address)
	}

	waitForRestart()

	app.info.Printf(lm.QuitReceived)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := SRV.Shutdown(ctx); err != nil {
		app.err.Fatalf(lm.FailedQuit, err)
	}
	app.info.Println(lm.Quit)
	return
}

func shutdown() {
	QUIT = true
	RESTART <- true
	// Safety Sleep (Ensure shutdown tasks get done)
	time.Sleep(time.Second)
}

func (app *appContext) shutdown() {
	app.info.Println(lm.Quitting)
	shutdown()
}

// Receives a restart signal and re-broadcasts it for other components.
func waitForRestart() {
	RESTARTLISTENERCOUNT++
	<-RESTART
	RESTARTLISTENERCOUNT--
	if RESTARTLISTENERCOUNT > 0 {
		RESTART <- true
	}
}

func flagPassed(name string) (found bool) {
	for i, f := range os.Args {
		if f == name {
			found = true
			// Remove the flag, to avoid issues wit the flag library.
			os.Args = append(os.Args[:i], os.Args[i+1:]...)
			return

		}
	}
	return
}

// @title jfa-go internal API
// @version 0.5.2
// @description API for the jfa-go frontend
// @contact.name Harvey Tindall
// @contact.email hrfee@hrfee.dev
// @license.name MIT
// @license.url https://raw.githubusercontent.com/hrfee/jfa-go/main/LICENSE
// @BasePath /

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization

// @securityDefinitions.basic getTokenAuth
// @name getTokenAuth

// @securityDefinitions.basic getUserTokenAuth
// @name getUserTokenAuth

// @tag.name Auth
// @tag.description -Get a token here if running swagger UI locally.-

// @tag.name User Page
// @tag.description User-page related routes.

// @tag.name Users
// @tag.description Jellyfin user related operations.

// @tag.name Invites
// @tag.description Invite related operations.

// @tag.name Profiles & Settings
// @tag.description Profile and settings related operations.

// @tag.name Activity
// @tag.description Routes related to the activity log.

// @tag.name Configuration
// @tag.description jfa-go settings.

// @tag.name Ombi
// @tag.description Ombi related operations.

// @tag.name Backups
// @tag.description Database backup/restore operations.

// @tag.name Other
// @tag.description Things that dont fit elsewhere.

// @tag.name Statistics
// @tag.description Routes that expose useful info/stats.

func printVersion() {
	tray := ""
	if TRAY {
		tray = " TrayIcon"
	}
	fmt.Println(info("jfa-go version: %s (%s)%s\n", hiwhite(version), white(commit), tray))
}

const SYSTEMD_SERVICE = "jfa-go.service"

func main() {
	// Generate list of "-tags" for about page.
	BuildTagsE2EE()
	BuildTagsTray()
	BuildTagsExternal()

	f, err := logOutput()
	if err != nil {
		fmt.Printf(lm.FailedLogging, err)
	}
	defer f()
	printVersion()
	SOCK = filepath.Join(temp, SOCK)
	fmt.Printf(lm.SocketPath+"\n", SOCK)
	if flagPassed("test") {
		TEST = true
	}
	executable, _ := os.Executable()
	loadFilesystems(filepath.Dir(executable), logger.NewLogger(os.Stdout, "[INFO] ", log.Ltime, color.FgHiWhite))

	quit := make(chan os.Signal, 0)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	// defer close(quit)
	go func() {
		<-quit
		shutdown()
	}()

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
			fmt.Printf(lm.FailedSocketConnect+"\n", SOCK, err)
			fmt.Println(lm.SocketCheckRunning)
			os.Exit(1)
		}
		_, err = con.Write([]byte("stop"))
		if err != nil {
			fmt.Printf(lm.FailedSocketWrite+"\n", SOCK, err)
			fmt.Println(lm.SocketCheckRunning)
			os.Exit(1)
		}
		fmt.Println(lm.SocketWrite)
	} else if flagPassed("daemon") {
		start(true, true)
	} else if flagPassed("systemd") {
		service, err := fs.ReadFile(localFS, SYSTEMD_SERVICE)
		if err != nil {
			fmt.Printf(lm.FailedReading+"\n", SYSTEMD_SERVICE, err)
			os.Exit(1)
		}
		absPath, err := os.Executable()
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
		err = os.WriteFile(SYSTEMD_SERVICE, service, 0666)
		if err != nil {
			fmt.Printf(lm.FailedWriting+"\n", SYSTEMD_SERVICE, err)
			os.Exit(1)
		}
		fmt.Println(info(`If you want to execute jfa-go with special arguments, re-run this command with them.
Move the newly created SYSTEMD_SERVICE file to ~/.config/systemd/user (Creating it if necessary).
Then run "systemctl --user daemon-reload".
You can then run:

`))
		// I have no idea why sleeps are necessary, but if not the lines print in the wrong order.
		time.Sleep(time.Millisecond)
		color.New(color.FgGreen).Print("To start: ")
		time.Sleep(time.Millisecond)
		fmt.Print(info("systemctl --user start jfa-go\n\n"))
		time.Sleep(time.Millisecond)
		color.New(color.FgRed).Print("To stop: ")
		time.Sleep(time.Millisecond)
		fmt.Print(info("systemctl --user stop jfa-go\n\n"))
		time.Sleep(time.Millisecond)
		color.New(color.FgYellow).Print("To restart: ")
		time.Sleep(time.Millisecond)
		fmt.Print(info("systemctl --user stop jfa-go\n"))
	} else if TRAY {
		RunTray()
	} else {
		RESTART = make(chan bool, 1)
		start(false, true)
		for {
			if QUIT {
				break
			}
			printVersion()
			start(false, false)
		}
	}
}
