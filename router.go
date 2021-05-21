package main

import (
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// loads HTML templates. If [files]/html_templates is set, alternative files inside the directory are loaded in place of the internal templates.
func (app *appContext) loadHTML(router *gin.Engine) {
	customPath := app.config.Section("files").Key("html_templates").MustString("")
	templatePath := "html"
	htmlFiles, err := fs.ReadDir(localFS, templatePath)
	if err != nil {
		app.err.Fatalf("Couldn't access template directory: \"%s\"", templatePath)
		return
	}
	loadFiles := make([]string, len(htmlFiles))
	for i, f := range htmlFiles {
		if _, err := os.Stat(filepath.Join(customPath, f.Name())); os.IsNotExist(err) {
			app.debug.Printf("Using default \"%s\"", f.Name())
			loadFiles[i] = FSJoin(templatePath, f.Name())
		} else {
			app.info.Printf("Using custom \"%s\"", f.Name())
			loadFiles[i] = filepath.Join(filepath.Join(customPath, f.Name()))
		}
	}
	tmpl, err := template.ParseFS(localFS, loadFiles...)
	if err != nil {
		app.err.Fatalf("Failed to load templates: %v", err)
	}
	router.SetHTMLTemplate(tmpl)
}

// sets gin logger.
func setGinLogger(router *gin.Engine, debugMode bool) {
	sprintf := color.New(color.Faint).SprintfFunc()
	if debugMode {
		router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return sprintf("[GIN/DEBUG] %s: %s(%s) => %d in %s; %s\n",
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
			return sprintf("[GIN] %s(%s) => %d\n",
				param.Method,
				param.Path,
				param.StatusCode,
			)
		}))
	}
}

func (app *appContext) loadRouter(address string, debug bool) *gin.Engine {
	if debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	setGinLogger(router, debug)

	router.Use(gin.Recovery())
	app.loadHTML(router)
	router.Use(static.Serve("/", app.webFS))
	router.NoRoute(app.NoRouteHandler)
	if *PPROF {
		app.debug.Println("Loading pprof")
		pprof.Register(router)
	}
	SRV = &http.Server{
		Addr:    address,
		Handler: router,
	}
	return router
}

func (app *appContext) loadRoutes(router *gin.Engine) {
	routePrefixes := []string{app.URLBase}
	if app.URLBase != "" {
		routePrefixes = append(routePrefixes, "")
	}
	for _, p := range routePrefixes {
		router.GET(p+"/lang/:page", app.GetLanguages)
		router.Use(static.Serve(p+"/", app.webFS))
		router.GET(p+"/", app.AdminPage)

		if app.config.Section("password_resets").Key("link_reset").MustBool(false) {
			router.GET(p+"/reset", app.ResetPassword)
		}

		router.GET(p+"/accounts", app.AdminPage)
		router.GET(p+"/settings", app.AdminPage)
		router.GET(p+"/lang/:page/:file", app.ServeLang)
		router.GET(p+"/token/login", app.getTokenLogin)
		router.GET(p+"/token/refresh", app.getTokenRefresh)
		router.POST(p+"/newUser", app.NewUser)
		router.Use(static.Serve(p+"/invite/", app.webFS))
		router.GET(p+"/invite/:invCode", app.InviteProxy)
		if telegramEnabled {
			router.GET(p+"/invite/:invCode/telegram/verified/:pin", app.TelegramVerifiedInvite)
		}
		if discordEnabled {
			router.GET(p+"/invite/:invCode/discord/verified/:pin", app.DiscordVerifiedInvite)
		}
	}
	if *SWAGGER {
		app.info.Print(warning("\n\nWARNING: Swagger should not be used on a public instance.\n\n"))
		for _, p := range routePrefixes {
			router.GET(p+"/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}
	}
	api := router.Group("/", app.webAuth())
	for _, p := range routePrefixes {
		router.POST(p+"/logout", app.Logout)
		api.DELETE(p+"/users", app.DeleteUsers)
		api.GET(p+"/users", app.GetUsers)
		api.POST(p+"/users", app.NewUserAdmin)
		api.POST(p+"/users/extend", app.ExtendExpiry)
		api.POST(p+"/users/enable", app.EnableDisableUsers)
		api.POST(p+"/invites", app.GenerateInvite)
		api.GET(p+"/invites", app.GetInvites)
		api.DELETE(p+"/invites", app.DeleteInvite)
		api.POST(p+"/invites/profile", app.SetProfile)
		api.GET(p+"/profiles", app.GetProfiles)
		api.POST(p+"/profiles/default", app.SetDefaultProfile)
		api.POST(p+"/profiles", app.CreateProfile)
		api.DELETE(p+"/profiles", app.DeleteProfile)
		api.POST(p+"/invites/notify", app.SetNotify)
		api.POST(p+"/users/emails", app.ModifyEmails)
		// api.POST(p + "/setDefaults", app.SetDefaults)
		api.POST(p+"/users/settings", app.ApplySettings)
		api.POST(p+"/users/announce", app.Announce)
		api.GET(p+"/config/update", app.CheckUpdate)
		api.POST(p+"/config/update", app.ApplyUpdate)
		api.GET(p+"/config/emails", app.GetCustomEmails)
		api.GET(p+"/config/emails/:id", app.GetCustomEmailTemplate)
		api.POST(p+"/config/emails/:id", app.SetCustomEmail)
		api.POST(p+"/config/emails/:id/state/:state", app.SetCustomEmailState)
		api.GET(p+"/config", app.GetConfig)
		api.POST(p+"/config", app.ModifyConfig)
		api.POST(p+"/restart", app.restart)
		if telegramEnabled || discordEnabled {
			api.GET(p+"/telegram/pin", app.TelegramGetPin)
			api.GET(p+"/telegram/verified/:pin", app.TelegramVerified)
			api.POST(p+"/users/telegram", app.TelegramAddUser)
		}
		if discordEnabled || telegramEnabled {
			api.POST(p+"/users/contact", app.SetContactMethods)
		}
		if app.config.Section("ombi").Key("enabled").MustBool(false) {
			api.GET(p+"/ombi/users", app.OmbiUsers)
			api.POST(p+"/ombi/defaults", app.SetOmbiDefaults)
		}
	}
}

func (app *appContext) loadSetup(router *gin.Engine) {
	router.GET("/lang/:page", app.GetLanguages)
	router.GET("/", app.ServeSetup)
	router.POST("/jellyfin/test", app.TestJF)
	router.POST("/config", app.ModifyConfig)
}
