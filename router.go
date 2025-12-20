package main

import (
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	// Disables authentication for the API. Do not use!
	NO_API_AUTH_DO_NOT_USE = false
	NO_API_AUTH_FORCE_JFID = ""
)

// loads HTML templates. If [files]/html_templates is set, alternative files inside the directory are loaded in place of the internal templates.
func (app *appContext) loadHTML(router *gin.Engine) {
	customPath := app.config.Section("files").Key("html_templates").MustString("")
	templatePath := "html"
	htmlFiles, err := fs.ReadDir(localFS, templatePath)
	if err != nil {
		app.err.Fatalf(lm.FailedReading, templatePath, err)
		return
	}
	loadInternal := []string{}
	loadExternal := []string{}
	for _, f := range htmlFiles {
		if _, err := os.Stat(filepath.Join(customPath, f.Name())); os.IsNotExist(err) {
			app.debug.Printf(lm.UseDefaultHTML, f.Name())
			loadInternal = append(loadInternal, FSJoin(templatePath, f.Name()))
		} else {
			app.info.Printf(lm.UseCustomHTML, f.Name())
			loadExternal = append(loadExternal, filepath.Join(filepath.Join(customPath, f.Name())))
		}
	}
	var tmpl *template.Template
	if len(loadInternal) != 0 {
		tmpl, err = template.ParseFS(localFS, loadInternal...)
		if err != nil {
			app.err.Fatalf(lm.FailedLoadTemplates, lm.Internal, err)
		}
	}
	if len(loadExternal) != 0 {
		tmpl, err = tmpl.ParseFiles(loadExternal...)
		if err != nil {
			app.err.Fatalf(lm.FailedLoadTemplates, lm.External, err)
		}
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
	router.Use(serveTaggedStatic("/", app.webFS))
	router.NoRoute(app.NoRouteHandler)
	if *PPROF {
		app.debug.Println(lm.RegisterPprof)
		pprof.Register(router)
	}
	SRV = &http.Server{
		Addr:    address,
		Handler: router,
	}
	return router
}

func (app *appContext) loadRoutes(router *gin.Engine) {
	routePrefixes := []string{PAGES.Base}
	if PAGES.Base != "" {
		routePrefixes = append(routePrefixes, "")
	}

	userPageEnabled := app.config.Section("user_page").Key("enabled").MustBool(true) && app.config.Section("ui").Key("jellyfin_login").MustBool(true)

	// Route collision may occur when reverse proxy subfolder is the same as a pseudo-path (e.g. /accounts, /activity...). For non-obvious ones, recover from the panic.
	defer func() {
		if r := recover(); r != nil {
			app.err.Fatalf(lm.RouteCollision, PAGES.Base, r)
		}
	}()

	for _, p := range routePrefixes {
		router.GET(p+"/lang/:page", app.GetLanguages)
		router.Use(serveTaggedStatic(p+"/", app.webFS))
		router.GET(p+PAGES.Admin, app.AdminPage)

		if app.config.Section("password_resets").Key("link_reset").MustBool(false) {
			router.GET(p+"/reset", app.ResetPassword)
			if app.config.Section("password_resets").Key("set_password").MustBool(false) {
				router.POST(p+"/reset", app.ResetSetPassword)
			}
		}

		// Handle the obvious collision of /accounts
		if len(routePrefixes) == 1 || p != "" || PAGES.Admin != "" {
			router.GET(p+PAGES.Admin+"/accounts", app.AdminPage)
		}
		router.GET(p+PAGES.Admin+"/settings", app.AdminPage)
		router.GET(p+PAGES.Admin+"/activity", app.AdminPage)
		router.GET(p+PAGES.Admin+"/accounts/user/:userID", app.AdminPage)
		router.GET(p+PAGES.Admin+"/invites/:code", app.AdminPage)
		router.GET(p+"/lang/:page/:file", app.ServeLang)
		router.GET(p+"/token/login", app.getTokenLogin)
		router.GET(p+"/token/refresh", app.getTokenRefresh)
		router.POST(p+"/user/invite", app.NewUserFromInvite)
		router.Use(serveTaggedStatic(p+PAGES.Form+"/", app.webFS))
		router.GET(p+PAGES.Form+"/:invCode", app.InviteProxy)
		if app.config.Section("captcha").Key("enabled").MustBool(false) {
			router.GET(p+"/captcha/gen/:invCode", app.GenCaptcha)
			router.GET(p+"/captcha/img/:invCode/:captchaID", app.GetCaptcha)
			router.POST(p+"/captcha/verify/:invCode/:captchaID/:text", app.VerifyCaptcha)
		}
		if telegramEnabled {
			router.GET(p+PAGES.Form+"/:invCode/telegram/verified/:pin", app.TelegramVerifiedInvite)
		}
		if discordEnabled {
			router.GET(p+PAGES.Form+"/:invCode/discord/verified/:pin", app.DiscordVerifiedInvite)
			if app.config.Section("discord").Key("provide_invite").MustBool(false) {
				router.GET(p+PAGES.Form+"/:invCode/discord/invite", app.DiscordServerInvite)
			}
		}
		if matrixEnabled {
			router.GET(p+PAGES.Form+"/:invCode/matrix/verified/:userID/:pin", app.MatrixCheckPIN)
			router.POST(p+PAGES.Form+"/:invCode/matrix/user", app.MatrixSendPIN)
			router.POST(p+"/users/matrix", app.MatrixConnect)
		}
		if userPageEnabled {
			router.GET(p+PAGES.MyAccount, app.MyUserPage)
			router.GET(p+PAGES.MyAccount+"/password/reset", app.MyUserPage)
			router.GET(p+"/my/token/login", app.getUserTokenLogin)
			router.GET(p+"/my/token/refresh", app.getUserTokenRefresh)
			router.GET(p+"/my/confirm/:jwt", app.ConfirmMyAction)
			router.POST(p+"/my/password/reset/:address", app.ResetMyPassword)
		}
	}
	if *SWAGGER {
		app.info.Print(warning(lm.SwaggerWarning))
		for _, p := range routePrefixes {
			router.GET(p+"/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}
	}

	var api *gin.RouterGroup
	api = router.Group("/", app.webAuth())

	for _, p := range routePrefixes {
		var user *gin.RouterGroup
		if userPageEnabled {
			user = router.Group(p+"/my", app.userAuth())
		}
		router.POST(p+"/logout", app.Logout)
		api.DELETE(p+"/users", app.DeleteUsers)
		api.GET(p+"/users", app.GetUsers)
		api.GET(p+"/users/count", app.GetUserCount)
		api.POST(p+"/users", app.SearchUsers)
		api.POST(p+"/users/count", app.GetFilteredUserCount)
		api.GET(p+"/users/labels", app.GetLabels)
		api.POST(p+"/user", app.NewUserFromAdmin)
		api.POST(p+"/users/extend", app.ExtendExpiry)
		api.DELETE(p+"/users/:id/expiry", app.RemoveExpiry)
		api.GET(p+"/users/:id/activities/jellyfin", app.GetJFActivitesForUser)
		api.GET(p+"/users/:id/activities/jellyfin/count", app.CountJFActivitesForUser)
		api.POST(p+"/users/:id/activities/jellyfin", app.GetPaginatedJFActivitesForUser)
		api.POST(p+"/users/enable", app.EnableDisableUsers)
		api.POST(p+"/invites", app.GenerateInvite)
		api.GET(p+"/invites", app.GetInvites)
		api.GET(p+"/invites/count", app.GetInviteCount)
		api.GET(p+"/invites/count/used", app.GetInviteUsedCount)
		api.DELETE(p+"/invites", app.DeleteInvite)
		api.POST(p+"/invites/send", app.SendInvite)
		api.PATCH(p+"/invites/edit", app.EditInvite)
		api.GET(p+"/profiles", app.GetProfiles)
		api.GET(p+"/profiles/names", app.GetProfileNames)
		api.GET(p+"/profiles/raw/:name", app.GetRawProfile)
		api.PUT(p+"/profiles/raw/:name", app.ReplaceRawProfile)
		api.POST(p+"/profiles/default", app.SetDefaultProfile)
		api.POST(p+"/profiles", app.CreateProfile)
		api.DELETE(p+"/profiles", app.DeleteProfile)
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
		api.GET(p+"/tasks", app.TaskList)
		api.POST(p+"/tasks/housekeeping", app.TaskHousekeeping)
		api.POST(p+"/tasks/users", app.TaskUserCleanup)
		if app.config.Section("jellyseerr").Key("enabled").MustBool(false) {
			api.POST(p+"/tasks/jellyseerr", app.TaskJellyseerrImport)
		}
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
		if app.config.Section("jellyseerr").Key("enabled").MustBool(false) {
			api.GET(p+"/jellyseerr/users", app.JellyseerrUsers)
			api.POST(p+"/profiles/jellyseerr/:profile/:id", app.SetJellyseerrProfile)
			api.DELETE(p+"/profiles/jellyseerr/:profile", app.DeleteJellyseerrProfile)
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
		api.POST(p+"/activity/count", app.GetFilteredActivityCount)

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
