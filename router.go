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

	userPageEnabled := app.config.Section("user_page").Key("enabled").MustBool(true) && app.config.Section("ui").Key("jellyfin_login").MustBool(true)

	for _, p := range routePrefixes {
		router.GET(p+"/lang/:page", app.GetLanguages)
		router.Use(static.Serve(p+"/", app.webFS))
		router.GET(p+"/", app.AdminPage)

		if app.config.Section("password_resets").Key("link_reset").MustBool(false) {
			router.GET(p+"/reset", app.ResetPassword)
			if app.config.Section("password_resets").Key("set_password").MustBool(false) {
				router.POST(p+"/reset", app.ResetSetPassword)
			}
		}

		router.GET(p+"/accounts", app.AdminPage)
		router.GET(p+"/settings", app.AdminPage)
		router.GET(p+"/activity", app.AdminPage)
		router.GET(p+"/accounts/user/:userID", app.AdminPage)
		router.GET(p+"/invites/:code", app.AdminPage)
		router.GET(p+"/lang/:page/:file", app.ServeLang)
		router.GET(p+"/token/login", app.getTokenLogin)
		router.GET(p+"/token/refresh", app.getTokenRefresh)
		router.POST(p+"/newUser", app.NewUser)
		router.Use(static.Serve(p+"/invite/", app.webFS))
		router.GET(p+"/invite/:invCode", app.InviteProxy)
		if app.config.Section("captcha").Key("enabled").MustBool(false) {
			router.GET(p+"/captcha/gen/:invCode", app.GenCaptcha)
			router.GET(p+"/captcha/img/:invCode/:captchaID", app.GetCaptcha)
			router.POST(p+"/captcha/verify/:invCode/:captchaID/:text", app.VerifyCaptcha)
		}
		if telegramEnabled {
			router.GET(p+"/invite/:invCode/telegram/verified/:pin", app.TelegramVerifiedInvite)
		}
		if discordEnabled {
			router.GET(p+"/invite/:invCode/discord/verified/:pin", app.DiscordVerifiedInvite)
			if app.config.Section("discord").Key("provide_invite").MustBool(false) {
				router.GET(p+"/invite/:invCode/discord/invite", app.DiscordServerInvite)
			}
		}
		if matrixEnabled {
			router.GET(p+"/invite/:invCode/matrix/verified/:userID/:pin", app.MatrixCheckPIN)
			router.POST(p+"/invite/:invCode/matrix/user", app.MatrixSendPIN)
			router.POST(p+"/users/matrix", app.MatrixConnect)
		}
		if userPageEnabled {
			router.GET(p+"/my/account", app.MyUserPage)
			router.GET(p+"/my/token/login", app.getUserTokenLogin)
			router.GET(p+"/my/token/refresh", app.getUserTokenRefresh)
			router.GET(p+"/my/confirm/:jwt", app.ConfirmMyAction)
			router.POST(p+"/my/password/reset/:address", app.ResetMyPassword)
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
		var user *gin.RouterGroup
		if userPageEnabled {
			user = router.Group(p+"/my", app.userAuth())
		}
		router.POST(p+"/logout", app.Logout)
		api.DELETE(p+"/users", app.DeleteUsers)
		api.GET(p+"/users", app.GetUsers)
		api.POST(p+"/users", app.NewUserAdmin)
		api.POST(p+"/users/extend", app.ExtendExpiry)
		api.DELETE(p+"/users/:id/expiry", app.RemoveExpiry)
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
		api.POST(p+"/users/labels", app.ModifyLabels)
		api.POST(p+"/users/accounts-admin", app.SetAccountsAdmin)
		// api.POST(p + "/setDefaults", app.SetDefaults)
		api.POST(p+"/users/settings", app.ApplySettings)
		api.POST(p+"/users/announce", app.Announce)

		api.GET(p+"/users/announce", app.GetAnnounceTemplates)
		api.POST(p+"/users/announce/template", app.SaveAnnounceTemplate)
		api.GET(p+"/users/announce/:name", app.GetAnnounceTemplate)
		api.DELETE(p+"/users/announce/:name", app.DeleteAnnounceTemplate)

		api.POST(p+"/users/password-reset", app.AdminPasswordReset)

		api.GET(p+"/config/update", app.CheckUpdate)
		api.POST(p+"/config/update", app.ApplyUpdate)
		api.GET(p+"/config/emails", app.GetCustomContent)
		api.GET(p+"/config/emails/:id", app.GetCustomMessageTemplate)
		api.POST(p+"/config/emails/:id", app.SetCustomMessage)
		api.POST(p+"/config/emails/:id/state/:state", app.SetCustomMessageState)
		api.GET(p+"/config", app.GetConfig)
		api.POST(p+"/config", app.ModifyConfig)
		api.POST(p+"/restart", app.restart)
		api.GET(p+"/logs", app.GetLog)
		api.POST(p+"/backups", app.CreateBackup)
		api.GET(p+"/backups/:fname", app.GetBackup)
		api.GET(p+"/backups", app.GetBackups)
		api.POST(p+"/backups/restore/:fname", app.RestoreLocalBackup)
		api.POST(p+"/backups/restore", app.RestoreBackup)
		if telegramEnabled || discordEnabled || matrixEnabled {
			api.GET(p+"/telegram/pin", app.TelegramGetPin)
			api.GET(p+"/telegram/verified/:pin", app.TelegramVerified)
			api.POST(p+"/users/telegram", app.TelegramAddUser)
			api.DELETE(p+"/users/telegram", app.UnlinkTelegram)
			api.DELETE(p+"/users/discord", app.UnlinkDiscord)
			api.DELETE(p+"/users/matrix", app.UnlinkMatrix)
		}
		if emailEnabled {
			api.POST(p+"/users/contact", app.SetContactMethods)
		}
		if discordEnabled {
			api.GET(p+"/users/discord/:username", app.DiscordGetUsers)
			api.POST(p+"/users/discord", app.DiscordConnect)
		}
		if app.config.Section("ombi").Key("enabled").MustBool(false) {
			api.GET(p+"/ombi/users", app.OmbiUsers)
			api.POST(p+"/profiles/ombi/:profile", app.SetOmbiProfile)
			api.DELETE(p+"/profiles/ombi/:profile", app.DeleteOmbiProfile)
		}
		api.POST(p+"/matrix/login", app.MatrixLogin)
		if app.config.Section("user_page").Key("referrals").MustBool(false) {
			api.POST(p+"/users/referral/:mode/:source/:useExpiry", app.EnableReferralForUsers)
			api.DELETE(p+"/users/referral", app.DisableReferralForUsers)
			api.POST(p+"/profiles/referral/:profile/:invite/:useExpiry", app.EnableReferralForProfile)
			api.DELETE(p+"/profiles/referral/:profile", app.DisableReferralForProfile)
		}

		api.POST(p+"/activity", app.GetActivities)
		api.DELETE(p+"/activity/:id", app.DeleteActivity)
		api.GET(p+"/activity/count", app.GetActivityCount)

		if userPageEnabled {
			user.GET("/details", app.MyDetails)
			user.POST("/contact", app.SetMyContactMethods)
			user.POST("/logout", app.LogoutUser)
			user.POST("/email", app.ModifyMyEmail)
			user.GET("/discord/invite", app.MyDiscordServerInvite)
			user.GET("/pin/:service", app.GetMyPIN)
			user.GET("/discord/verified/:pin", app.MyDiscordVerifiedInvite)
			user.GET("/telegram/verified/:pin", app.MyTelegramVerifiedInvite)
			user.POST("/matrix/user", app.MatrixSendMyPIN)
			user.GET("/matrix/verified/:userID/:pin", app.MatrixCheckMyPIN)
			user.DELETE("/discord", app.UnlinkMyDiscord)
			user.DELETE("/telegram", app.UnlinkMyTelegram)
			user.DELETE("/matrix", app.UnlinkMyMatrix)
			user.POST("/password", app.ChangeMyPassword)
			if app.config.Section("user_page").Key("referrals").MustBool(false) {
				user.GET("/referral", app.GetMyReferral)
			}
		}
	}
}

func (app *appContext) loadSetup(router *gin.Engine) {
	router.GET("/lang/:page", app.GetLanguages)
	router.GET("/", app.ServeSetup)
	router.POST("/jellyfin/test", app.TestJF)
	router.POST("/config", app.ModifyConfig)
}
