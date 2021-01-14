package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func gcHTML(gc *gin.Context, code int, file string, templ gin.H) {
	gc.Header("Cache-Control", "no-cache")
	gc.HTML(code, file, templ)
}

func (app *appContext) AdminPage(gc *gin.Context) {
	lang := gc.Query("lang")
	if lang == "" {
		lang = app.storage.lang.chosenAdminLang
	} else if _, ok := app.storage.lang.Form[lang]; !ok {
		lang = app.storage.lang.chosenAdminLang
	}
	emailEnabled, _ := app.config.Section("invite_emails").Key("enabled").Bool()
	notificationsEnabled, _ := app.config.Section("notifications").Key("enabled").Bool()
	ombiEnabled := app.config.Section("ombi").Key("enabled").MustBool(false)
	if pusher := gc.Writer.Pusher(); pusher != nil {
		toPush := []string{"/js/admin.js", "/js/theme.js", "/js/lang.js", "/js/modal.js", "/js/tabs.js", "/js/invites.js", "/js/accounts.js", "/js/settings.js", "/js/profiles.js", "/js/common.js"}
		for _, f := range toPush {
			if err := pusher.Push(f, nil); err != nil {
				app.debug.Printf("Failed HTTP2 ServerPush of \"%s\": %+v", f, err)
			}
		}
	}
	gcHTML(gc, http.StatusOK, "admin.html", gin.H{
		"urlBase":         app.URLBase,
		"cssClass":        app.cssClass,
		"contactMessage":  "",
		"email_enabled":   emailEnabled,
		"notifications":   notificationsEnabled,
		"version":         VERSION,
		"commit":          COMMIT,
		"ombiEnabled":     ombiEnabled,
		"username":        !app.config.Section("email").Key("no_username").MustBool(false),
		"strings":         app.storage.lang.Admin[lang]["strings"],
		"quantityStrings": app.storage.lang.Admin[lang]["quantityStrings"],
		"language":        app.storage.lang.AdminJSON[lang],
	})
}

func (app *appContext) InviteProxy(gc *gin.Context) {
	code := gc.Param("invCode")
	lang := gc.Query("lang")
	if lang == "" {
		lang = app.storage.lang.chosenFormLang
	} else if _, ok := app.storage.lang.Form[lang]; !ok {
		lang = app.storage.lang.chosenFormLang
	}
	/* Don't actually check if the invite is valid, just if it exists, just so the page loads quicker. Invite is actually checked on submit anyway. */
	// if app.checkInvite(code, false, "") {
	if _, ok := app.storage.invites[code]; ok {
		email := app.storage.invites[code].Email
		if strings.Contains(email, "Failed") {
			email = ""
		}
		gcHTML(gc, http.StatusOK, "form-loader.html", gin.H{
			"urlBase":        app.URLBase,
			"cssClass":       app.cssClass,
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
			"helpMessage":    app.config.Section("ui").Key("help_message").String(),
			"successMessage": app.config.Section("ui").Key("success_message").String(),
			"jfLink":         app.config.Section("jellyfin").Key("public_server").String(),
			"validate":       app.config.Section("password_validation").Key("enabled").MustBool(false),
			"requirements":   app.validator.getCriteria(),
			"email":          email,
			"username":       !app.config.Section("email").Key("no_username").MustBool(false),
			"lang":           app.storage.lang.Form[lang]["strings"],
			"code":           code,
		})
	} else {
		gcHTML(gc, 404, "invalidCode.html", gin.H{
			"cssClass":       app.cssClass,
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		})
	}
}

func (app *appContext) NoRouteHandler(gc *gin.Context) {
	gcHTML(gc, 404, "404.html", gin.H{
		"cssClass":       app.cssClass,
		"contactMessage": app.config.Section("ui").Key("contact_message").String(),
	})
}
