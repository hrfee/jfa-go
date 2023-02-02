package main

import (
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/hrfee/mediabrowser"
	"github.com/steambap/captcha"
)

var cssVersion string
var css = []string{cssVersion + "bundle.css", "remixicon.css"}
var cssHeader string

func (app *appContext) loadCSSHeader() string {
	l := len(css)
	h := ""
	for i, f := range css {
		h += "<" + app.URLBase + "/css/" + f + ">; rel=preload; as=style"
		if l > 1 && i != (l-1) {
			h += ", "
		}
	}
	return h
}

func (app *appContext) getURLBase(gc *gin.Context) string {
	if strings.HasPrefix(gc.Request.URL.String(), app.URLBase) {
		return app.URLBase
	}
	return ""
}

func gcHTML(gc *gin.Context, code int, file string, templ gin.H) {
	gc.Header("Cache-Control", "no-cache")
	gc.HTML(code, file, templ)
}

func (app *appContext) pushResources(gc *gin.Context, admin bool) {
	if pusher := gc.Writer.Pusher(); pusher != nil {
		app.debug.Println("Using HTTP2 Server push")
		if admin {
			toPush := []string{"/js/admin.js", "/js/theme.js", "/js/lang.js", "/js/modal.js", "/js/tabs.js", "/js/invites.js", "/js/accounts.js", "/js/settings.js", "/js/profiles.js", "/js/common.js"}
			for _, f := range toPush {
				if err := pusher.Push(app.URLBase+f, nil); err != nil {
					app.debug.Printf("Failed HTTP2 ServerPush of \"%s\": %+v", f, err)
				}
			}
		}
	}
	gc.Header("Link", cssHeader)
}

type Page int

const (
	AdminPage Page = iota + 1
	FormPage
	PWRPage
)

func (app *appContext) getLang(gc *gin.Context, page Page, chosen string) string {
	lang := gc.Query("lang")
	cookie, err := gc.Cookie("lang")
	if lang != "" {
		switch page {
		case AdminPage:
			if _, ok := app.storage.lang.Admin[lang]; ok {
				gc.SetCookie("lang", lang, (365 * 3600), "/", gc.Request.URL.Hostname(), true, true)
				return lang
			}
		case FormPage:
			if _, ok := app.storage.lang.Form[lang]; ok {
				gc.SetCookie("lang", lang, (365 * 3600), "/", gc.Request.URL.Hostname(), true, true)
				return lang
			}
		case PWRPage:
			if _, ok := app.storage.lang.PasswordReset[lang]; ok {
				gc.SetCookie("lang", lang, (365 * 3600), "/", gc.Request.URL.Hostname(), true, true)
				return lang
			}
		}
	}
	if cookie != "" && err == nil {
		switch page {
		case AdminPage:
			if _, ok := app.storage.lang.Admin[cookie]; ok {
				return cookie
			}
		case FormPage:
			if _, ok := app.storage.lang.Form[cookie]; ok {
				return cookie
			}
		case PWRPage:
			if _, ok := app.storage.lang.PasswordReset[cookie]; ok {
				return cookie
			}
		}
	}
	return chosen
}

func (app *appContext) AdminPage(gc *gin.Context) {
	app.pushResources(gc, true)
	lang := app.getLang(gc, AdminPage, app.storage.lang.chosenAdminLang)
	emailEnabled, _ := app.config.Section("invite_emails").Key("enabled").Bool()
	notificationsEnabled, _ := app.config.Section("notifications").Key("enabled").Bool()
	ombiEnabled := app.config.Section("ombi").Key("enabled").MustBool(false)
	jfAdminOnly := app.config.Section("ui").Key("admin_only").MustBool(true)
	jfAllowAll := app.config.Section("ui").Key("allow_all").MustBool(false)
	var license string
	l, err := fs.ReadFile(localFS, "LICENSE")
	if err != nil {
		app.debug.Printf("Failed to load LICENSE: %s", err)
		license = ""
	}
	license = string(l)
	gcHTML(gc, http.StatusOK, "admin.html", gin.H{
		"urlBase":          app.getURLBase(gc),
		"cssClass":         app.cssClass,
		"cssVersion":       cssVersion,
		"contactMessage":   "",
		"emailEnabled":     emailEnabled,
		"telegramEnabled":  telegramEnabled,
		"discordEnabled":   discordEnabled,
		"matrixEnabled":    matrixEnabled,
		"ombiEnabled":      ombiEnabled,
		"linkResetEnabled": app.config.Section("password_resets").Key("link_reset").MustBool(false),
		"notifications":    notificationsEnabled,
		"version":          version,
		"commit":           commit,
		"username":         !app.config.Section("email").Key("no_username").MustBool(false),
		"strings":          app.storage.lang.Admin[lang].Strings,
		"quantityStrings":  app.storage.lang.Admin[lang].QuantityStrings,
		"language":         app.storage.lang.Admin[lang].JSON,
		"langName":         lang,
		"license":          license,
		"jellyfinLogin":    app.jellyfinLogin,
		"jfAdminOnly":      jfAdminOnly,
		"jfAllowAll":       jfAllowAll,
	})
}

func (app *appContext) ResetPassword(gc *gin.Context) {
	isBot := strings.Contains(gc.Request.Header.Get("User-Agent"), "Bot")
	setPassword := app.config.Section("password_resets").Key("set_password").MustBool(false)
	pin := gc.Query("pin")
	if pin == "" {
		app.NoRouteHandler(gc)
		return
	}
	app.pushResources(gc, false)
	lang := app.getLang(gc, PWRPage, app.storage.lang.chosenPWRLang)
	data := gin.H{
		"urlBase":        app.getURLBase(gc),
		"cssClass":       app.cssClass,
		"cssVersion":     cssVersion,
		"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		"strings":        app.storage.lang.PasswordReset[lang].Strings,
		"success":        false,
		"ombiEnabled":    app.config.Section("ombi").Key("enabled").MustBool(false),
	}
	pwr, isInternal := app.internalPWRs[pin]
	if isInternal && setPassword {
		data["helpMessage"] = app.config.Section("ui").Key("help_message").String()
		data["successMessage"] = app.config.Section("ui").Key("success_message").String()
		data["jfLink"] = app.config.Section("ui").Key("redirect_url").String()
		data["redirectToJellyfin"] = app.config.Section("ui").Key("auto_redirect").MustBool(false)
		data["validate"] = app.config.Section("password_validation").Key("enabled").MustBool(false)
		data["requirements"] = app.validator.getCriteria()
		data["strings"] = app.storage.lang.PasswordReset[lang].Strings
		data["validationStrings"] = app.storage.lang.Form[lang].validationStringsJSON
		data["notifications"] = app.storage.lang.Form[lang].notificationsJSON
		data["langName"] = lang
		data["passwordReset"] = true
		data["telegramEnabled"] = false
		data["discordEnabled"] = false
		data["matrixEnabled"] = false
		gcHTML(gc, http.StatusOK, "form-loader.html", data)
		return
	}
	defer gcHTML(gc, http.StatusOK, "password-reset.html", data)
	// If it's a bot, pretend to be a success so the preview is nice.
	if isBot {
		app.debug.Println("PWR: Ignoring magic link visit from bot")
		data["success"] = true
		data["pin"] = "NO-BO-TS"
		return
	}
	// if reset, ok := app.internalPWRs[pin]; ok {
	// 	status, err := app.jf.ResetPasswordAdmin(reset.ID)
	// 	if !(status == 200 || status == 204) || err != nil {
	// 		app.err.Printf("Password Reset failed (%d): %v", status, err)
	// 		return
	// 	}
	// 	status, err = app.jf.SetPassword(reset.ID, "", pin)
	// 	if !(status == 200 || status == 204) || err != nil {
	// 		app.err.Printf("Password Reset failed (%d): %v", status, err)
	// 		return
	// 	}
	// 	data["success"] = true
	// 	data["pin"] = pin
	// }
	var resp mediabrowser.PasswordResetResponse
	var status int
	var err error
	var username string
	if !isInternal {
		resp, status, err = app.jf.ResetPassword(pin)
	} else if time.Now().After(pwr.Expiry) {
		app.debug.Printf("Ignoring PWR request due to expired internal PIN: %s", pin)
		app.NoRouteHandler(gc)
		return
	} else {
		status, err = app.jf.ResetPasswordAdmin(pwr.ID)
		if !(status == 200 || status == 204) || err != nil {
			app.err.Printf("Password Reset failed (%d): %v", status, err)
		} else {
			status, err = app.jf.SetPassword(pwr.ID, "", pin)
		}
		username = pwr.Username
	}
	if (status == 200 || status == 204) && err == nil && (isInternal || resp.Success) {
		data["success"] = true
		data["pin"] = pin
		if !isInternal {
			username = resp.UsersReset[0]
		}
	} else {
		app.err.Printf("Password Reset failed (%d): %v", status, err)
	}
	if app.config.Section("ombi").Key("enabled").MustBool(false) {
		jfUser, status, err := app.jf.UserByName(username, false)
		if status != 200 || err != nil {
			app.err.Printf("Failed to get user \"%s\" from jellyfin/emby (%d): %v", username, status, err)
			return
		}
		ombiUser, status, err := app.getOmbiUser(jfUser.ID)
		if status != 200 || err != nil {
			app.err.Printf("Failed to get user \"%s\" from ombi (%d): %v", username, status, err)
			return
		}
		ombiUser["password"] = pin
		status, err = app.ombi.ModifyUser(ombiUser)
		if status != 200 || err != nil {
			app.err.Printf("Failed to set password for ombi user \"%s\" (%d): %v", ombiUser["userName"], status, err)
			return
		}
		app.debug.Printf("Reset password for ombi user \"%s\"", ombiUser["userName"])
	}
}

// @Summary returns the captcha image corresponding to the given ID.
// @Param code path string true "invite code"
// @Param captchaID path string true "captcha ID"
// @Tags Other
// @Router /captcha/img/{code}/{captchaID} [get]
func (app *appContext) GetCaptcha(gc *gin.Context) {
	code := gc.Param("invCode")
	captchaID := gc.Param("captchaID")
	inv, ok := app.storage.invites[code]
	if !ok {
		gcHTML(gc, 404, "invalidCode.html", gin.H{
			"cssClass":       app.cssClass,
			"cssVersion":     cssVersion,
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		})
	}
	var capt *captcha.Data
	if inv.Captchas != nil {
		capt = inv.Captchas[captchaID]
	}
	if capt == nil {
		respondBool(400, false, gc)
		return
	}
	if err := capt.WriteImage(gc.Writer); err != nil {
		app.err.Printf("Failed to write CAPTCHA image: %v", err)
		respondBool(500, false, gc)
		return
	}
	gc.Status(200)
	return
}

// @Summary Generates a new captcha and returns it's ID. This can then be included in a request to /captcha/img/{id} to get an image.
// @Produce json
// @Param code path string true "invite code"
// @Success 200 {object} genCaptchaDTO
// @Router /captcha/gen/{code} [get]
// @Security Bearer
// @tags Users
func (app *appContext) GenCaptcha(gc *gin.Context) {
	code := gc.Param("invCode")
	inv, ok := app.storage.invites[code]
	if !ok {
		gcHTML(gc, 404, "invalidCode.html", gin.H{
			"cssClass":       app.cssClass,
			"cssVersion":     cssVersion,
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		})
	}
	capt, err := captcha.New(300, 100)
	if err != nil {
		app.err.Printf("Failed to generate captcha: %v", err)
		respondBool(500, false, gc)
		return
	}
	if inv.Captchas == nil {
		inv.Captchas = map[string]*captcha.Data{}
	}
	captchaID := genAuthToken()
	inv.Captchas[captchaID] = capt
	app.storage.invites[code] = inv
	app.storage.storeInvites()
	gc.JSON(200, genCaptchaDTO{captchaID})
	return
}

func (app *appContext) verifyCaptcha(code, id, text string) bool {
	inv, ok := app.storage.invites[code]
	if !ok || inv.Captchas == nil {
		app.debug.Printf("Couldn't find invite \"%s\"", code)
		return false
	}
	c, ok := inv.Captchas[id]
	if !ok {
		app.debug.Printf("Couldn't find Captcha \"%s\"", id)
		return false
	}
	return strings.ToLower(c.Text) == strings.ToLower(text)
}

// @Summary returns 204 if the given Captcha contents is correct for the corresponding captcha ID and invite code.
// @Param code path string true "invite code"
// @Param captchaID path string true "captcha ID"
// @Param text path string true "Captcha text"
// @Success 204
// @Tags Other
// @Router /captcha/verify/{code}/{captchaID}/{text} [get]
func (app *appContext) VerifyCaptcha(gc *gin.Context) {
	code := gc.Param("invCode")
	captchaID := gc.Param("captchaID")
	text := gc.Param("text")
	inv, ok := app.storage.invites[code]
	if !ok {
		gcHTML(gc, 404, "invalidCode.html", gin.H{
			"cssClass":       app.cssClass,
			"cssVersion":     cssVersion,
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		})
		return
	}
	var capt *captcha.Data
	if inv.Captchas != nil {
		capt = inv.Captchas[captchaID]
	}
	if capt == nil {
		respondBool(400, false, gc)
		return
	}
	if strings.ToLower(capt.Text) != strings.ToLower(text) {
		respondBool(400, false, gc)
		return
	}
	respondBool(204, true, gc)
	return
}

func (app *appContext) InviteProxy(gc *gin.Context) {
	app.pushResources(gc, false)
	code := gc.Param("invCode")
	lang := app.getLang(gc, FormPage, app.storage.lang.chosenFormLang)
	/* Don't actually check if the invite is valid, just if it exists, just so the page loads quicker. Invite is actually checked on submit anyway. */
	// if app.checkInvite(code, false, "") {
	inv, ok := app.storage.invites[code]
	if !ok {
		gcHTML(gc, 404, "invalidCode.html", gin.H{
			"cssClass":       app.cssClass,
			"cssVersion":     cssVersion,
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		})
		return
	}
	if key := gc.Query("key"); key != "" && app.config.Section("email_confirmation").Key("enabled").MustBool(false) {
		validKey := false
		keyIndex := -1
		for i, k := range inv.Keys {
			if k == key {
				validKey = true
				keyIndex = i
				break
			}
		}
		fail := func() {
			gcHTML(gc, 404, "404.html", gin.H{
				"cssClass":       app.cssClass,
				"cssVersion":     cssVersion,
				"contactMessage": app.config.Section("ui").Key("contact_message").String(),
			})
		}
		if !validKey {
			fail()
			return
		}
		token, err := jwt.Parse(key, checkToken)
		if err != nil {
			fail()
			app.err.Printf("Failed to parse key: %s", err)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		expiryUnix := int64(claims["exp"].(float64))
		if err != nil {
			fail()
			app.err.Printf("Failed to parse key expiry: %s", err)
			return
		}
		expiry := time.Unix(expiryUnix, 0)
		if !(ok && token.Valid && claims["invite"].(string) == code && claims["type"].(string) == "confirmation" && expiry.After(time.Now())) {
			fail()
			app.debug.Printf("Invalid key")
			return
		}
		req := newUserDTO{
			Email:    claims["email"].(string),
			Username: claims["username"].(string),
			Password: claims["password"].(string),
			Code:     claims["invite"].(string),
		}
		_, success := app.newUser(req, true)
		if !success {
			fail()
			return
		}
		jfLink := app.config.Section("ui").Key("redirect_url").String()
		if app.config.Section("ui").Key("auto_redirect").MustBool(false) {
			gc.Redirect(301, jfLink)
		} else {
			gcHTML(gc, http.StatusOK, "create-success.html", gin.H{
				"cssClass":       app.cssClass,
				"strings":        app.storage.lang.Form[lang].Strings,
				"successMessage": app.config.Section("ui").Key("success_message").String(),
				"contactMessage": app.config.Section("ui").Key("contact_message").String(),
				"jfLink":         jfLink,
			})
		}
		inv, ok := app.storage.invites[code]
		if ok {
			l := len(inv.Keys)
			inv.Keys[l-1], inv.Keys[keyIndex] = inv.Keys[keyIndex], inv.Keys[l-1]
			app.storage.invites[code] = inv
		}
		return
	}
	email := app.storage.invites[code].SendTo
	if strings.Contains(email, "Failed") || !strings.Contains(email, "@") {
		email = ""
	}
	telegram := telegramEnabled && app.config.Section("telegram").Key("show_on_reg").MustBool(true)
	discord := discordEnabled && app.config.Section("discord").Key("show_on_reg").MustBool(true)
	matrix := matrixEnabled && app.config.Section("matrix").Key("show_on_reg").MustBool(true)

	data := gin.H{
		"urlBase":            app.getURLBase(gc),
		"cssClass":           app.cssClass,
		"cssVersion":         cssVersion,
		"contactMessage":     app.config.Section("ui").Key("contact_message").String(),
		"helpMessage":        app.config.Section("ui").Key("help_message").String(),
		"successMessage":     app.config.Section("ui").Key("success_message").String(),
		"jfLink":             app.config.Section("ui").Key("redirect_url").String(),
		"redirectToJellyfin": app.config.Section("ui").Key("auto_redirect").MustBool(false),
		"validate":           app.config.Section("password_validation").Key("enabled").MustBool(false),
		"requirements":       app.validator.getCriteria(),
		"email":              email,
		"username":           !app.config.Section("email").Key("no_username").MustBool(false),
		"strings":            app.storage.lang.Form[lang].Strings,
		"validationStrings":  app.storage.lang.Form[lang].validationStringsJSON,
		"notifications":      app.storage.lang.Form[lang].notificationsJSON,
		"code":               code,
		"confirmation":       app.config.Section("email_confirmation").Key("enabled").MustBool(false),
		"userExpiry":         inv.UserExpiry,
		"userExpiryMonths":   inv.UserMonths,
		"userExpiryDays":     inv.UserDays,
		"userExpiryHours":    inv.UserHours,
		"userExpiryMinutes":  inv.UserMinutes,
		"userExpiryMessage":  app.storage.lang.Form[lang].Strings.get("yourAccountIsValidUntil"),
		"langName":           lang,
		"passwordReset":      false,
		"telegramEnabled":    telegram,
		"discordEnabled":     discord,
		"matrixEnabled":      matrix,
		"emailRequired":      app.config.Section("email").Key("required").MustBool(false),
		"captcha":            app.config.Section("captcha").Key("enabled").MustBool(false),
	}
	if telegram {
		data["telegramPIN"] = app.telegram.NewAuthToken()
		data["telegramUsername"] = app.telegram.username
		data["telegramURL"] = app.telegram.link
		data["telegramRequired"] = app.config.Section("telegram").Key("required").MustBool(false)
	}
	if matrix {
		data["matrixRequired"] = app.config.Section("matrix").Key("required").MustBool(false)
		data["matrixUser"] = app.matrix.userID
	}
	if discord {
		data["discordPIN"] = app.discord.NewAuthToken()
		data["discordUsername"] = app.discord.username
		data["discordRequired"] = app.config.Section("discord").Key("required").MustBool(false)
		data["discordSendPINMessage"] = template.HTML(app.storage.lang.Form[lang].Strings.template("sendPINDiscord", tmpl{
			"command":        `<span class="text-black dark:text-white font-mono">/` + app.config.Section("discord").Key("start_command").MustString("start") + `</span>`,
			"server_channel": app.discord.serverChannelName,
		}))
		data["discordServerName"] = app.discord.serverName
		data["discordInviteLink"] = app.discord.inviteChannelName != ""
	}

	// if discordEnabled {
	// 	pin := ""
	// 	for _, token := range app.discord.tokens {
	// 		if
	gcHTML(gc, http.StatusOK, "form-loader.html", data)
}

func (app *appContext) NoRouteHandler(gc *gin.Context) {
	app.pushResources(gc, false)
	gcHTML(gc, 404, "404.html", gin.H{
		"cssClass":       app.cssClass,
		"cssVersion":     cssVersion,
		"contactMessage": app.config.Section("ui").Key("contact_message").String(),
	})
}
