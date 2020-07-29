package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (ctx *appContext) AdminPage(gc *gin.Context) {
	bs5, _ := ctx.config.Section("ui").Key("bs5").Bool()
	emailEnabled, _ := ctx.config.Section("invite_emails").Key("enabled").Bool()
	notificationsEnabled, _ := ctx.config.Section("notifications").Key("enabled").Bool()
	gc.HTML(http.StatusOK, "admin.html", gin.H{
		"bs5":            bs5,
		"cssFile":        ctx.cssFile,
		"contactMessage": "",
		"email_enabled":  emailEnabled,
		"notifications":  notificationsEnabled,
	})
}
