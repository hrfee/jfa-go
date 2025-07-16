package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/gomarkdown/markdown"
	"github.com/hrfee/jfa-go/common"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/hrfee/mediabrowser"
	"github.com/lithammer/shortuuid/v3"
	"github.com/steambap/captcha"
)

var cssVersion string
var css = []string{cssVersion + "bundle.css", "remixicon.css"}
var cssHeader string

func (app *appContext) loadCSSHeader() string {
	l := len(css)
	h := ""
	for i, f := range css {
		h += "<" + PAGES.Base + "/css/" + f + ">; rel=preload; as=style"
		if l > 1 && i != (l-1) {
			h += ", "
		}
	}
	return h
}

func (app *appContext) getURLBase(gc *gin.Context) string {
	if strings.HasPrefix(gc.Request.URL.String(), PAGES.Base) {
		// Hack to fix the common URL base /accounts
		if PAGES.Base == "/accounts" && strings.HasPrefix(gc.Request.URL.String(), "/accounts/user/") {
			return ""
		}
		return PAGES.Base
	}
	return ""
}

func (app *appContext) gcHTML(gc *gin.Context, code int, file string, page Page, templ gin.H) {
	gc.Header("Cache-Control", "no-cache")
	app.BasePageTemplateValues(gc, page, templ)
	gc.HTML(code, file, templ)
}

func (app *appContext) pushResources(gc *gin.Context, page Page) {
	var toPush []string
	switch page {
	case AdminPage:
		toPush = []string{"/js/admin.js", "/js/theme.js", "/js/lang.js", "/js/modal.js", "/js/tabs.js", "/js/invites.js", "/js/accounts.js", "/js/settings.js", "/js/profiles.js", "/js/common.js"}
	case UserPage:
		toPush = []string{"/js/user.js", "/js/theme.js", "/js/lang.js", "/js/modal.js", "/js/common.js"}
	default:
		toPush = []string{}
	}
	urlBase := app.getURLBase(gc)
	if pusher := gc.Writer.Pusher(); pusher != nil {
		for _, f := range toPush {
			if err := pusher.Push(urlBase+f, nil); err != nil {
				app.debug.Printf(lm.FailedServerPush, err)
			}
		}
	}
	gc.Header("Link", cssHeader)
}

// Returns a gin.H with general values (url base, css version, etc.)
func (app *appContext) BasePageTemplateValues(gc *gin.Context, page Page, base gin.H) {
	set := func(k string, v any) {
		if _, ok := base[k]; !ok {
			base[k] = v
		}
	}

	pages := PagePathsDTO{
		PagePaths:   PAGES,
		ExternalURI: app.ExternalURI(gc),
		TrueBase:    PAGES.Base,
	}
	pages.Base = app.getURLBase(gc)
	switch page {
	case AdminPage:
		pages.Current = PAGES.Admin
	case FormPage:
		pages.Current = PAGES.Form
	case UserPage:
		pages.Current = PAGES.MyAccount
	default:
		pages.Current = "/"
	}
	set("pages", pages)
	ombiEnabled := app.config.Section("ombi").Key("enabled").MustBool(false)
	jellyseerrEnabled := app.config.Section("jellyseerr").Key("enabled").MustBool(false)
	notificationsEnabled, _ := app.config.Section("notifications").Key("enabled").Bool()
	set("notifications", notificationsEnabled)
	set("cssClass", app.cssClass)
	set("cssVersion", cssVersion)
	set("emailEnabled", emailEnabled)
	set("telegramEnabled", telegramEnabled)
	set("discordEnabled", discordEnabled)
	set("matrixEnabled", matrixEnabled)
	set("ombiEnabled", ombiEnabled)
	set("jellyseerrEnabled", jellyseerrEnabled)
	// QUIRK: The login modal html template uses this' existence to check if the modal is for the admin or user page.
	if page != AdminPage {
		set("pwrEnabled", app.config.Section("password_resets").Key("enabled").MustBool(false))
	}
	set("referralsEnabled", app.config.Section("user_page").Key("enabled").MustBool(false) && app.config.Section("user_page").Key("referrals").MustBool(false))
}

type Page int

const (
	AdminPage Page = iota + 1
	FormPage
	PWRPage
	UserPage
	OtherPage
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
		case FormPage, UserPage:
			if _, ok := app.storage.lang.User[lang]; ok {
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
		case FormPage, UserPage:
			if _, ok := app.storage.lang.User[cookie]; ok {
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
	app.pushResources(gc, AdminPage)

	// Pre-emptively (maybe) generate user cache
	go app.userCache.MaybeSync(app)

	lang := app.getLang(gc, AdminPage, app.storage.lang.chosenAdminLang)
	jfAdminOnly := app.config.Section("ui").Key("admin_only").MustBool(true)
	jfAllowAll := app.config.Section("ui").Key("allow_all").MustBool(false)
	var license string
	l, err := fs.ReadFile(localFS, "LICENSE")
	if err != nil {
		app.debug.Printf(lm.FailedReading, "LICENSE", err)
		license = ""
	}
	license = string(l)
	fontLicense, err := fs.ReadFile(localFS, filepath.Join("web", "fonts", "OFL.txt"))
	if err != nil {
		app.debug.Printf(lm.FailedReading, "fontLicense", err)
	}

	license += "---Hanken Grotesk---\n\n"
	license += string(fontLicense)

	if builtBy == "" {
		builtBy = "???"
	}

	app.gcHTML(gc, http.StatusOK, "admin.html", AdminPage, gin.H{
		"contactMessage":   "",
		"linkResetEnabled": app.config.Section("password_resets").Key("link_reset").MustBool(false),
		"version":          version,
		"commit":           commit,
		"buildTime":        buildTime,
		"builtBy":          builtBy,
		"buildTags":        buildTags,
		"username":         !app.config.Section("email").Key("no_username").MustBool(false),
		"strings":          app.storage.lang.Admin[lang].Strings,
		"quantityStrings":  app.storage.lang.Admin[lang].QuantityStrings,
		"language":         app.storage.lang.Admin[lang].JSON,
		"langName":         lang,
		"license":          license,
		"jellyfinLogin":    app.jellyfinLogin,
		"jfAdminOnly":      jfAdminOnly,
		"jfAllowAll":       jfAllowAll,
		"userPageEnabled":  app.config.Section("user_page").Key("enabled").MustBool(false),
		"showUserPageLink": app.config.Section("user_page").Key("show_link").MustBool(true),
		"loginAppearance":  app.config.Section("ui").Key("login_appearance").MustString("clear"),
	})
}

func (app *appContext) MyUserPage(gc *gin.Context) {
	app.pushResources(gc, UserPage)
	lang := app.getLang(gc, UserPage, app.storage.lang.chosenUserLang)
	data := gin.H{
		"contactMessage":    app.config.Section("ui").Key("contact_message").String(),
		"emailRequired":     app.config.Section("email").Key("required").MustBool(false),
		"linkResetEnabled":  app.config.Section("password_resets").Key("link_reset").MustBool(false),
		"username":          !app.config.Section("email").Key("no_username").MustBool(false),
		"strings":           app.storage.lang.User[lang].Strings,
		"validationStrings": app.storage.lang.User[lang].validationStringsJSON,
		"language":          app.storage.lang.User[lang].JSON,
		"langName":          lang,
		"jfLink":            app.config.Section("ui").Key("redirect_url").String(),
		"requirements":      app.validator.getCriteria(),
	}
	if telegramEnabled {
		data["telegramUsername"] = app.telegram.username
		data["telegramURL"] = app.telegram.link
		data["telegramRequired"] = app.config.Section("telegram").Key("required").MustBool(false)
	}
	if matrixEnabled {
		data["matrixRequired"] = app.config.Section("matrix").Key("required").MustBool(false)
		data["matrixUser"] = app.matrix.userID
	}
	if discordEnabled {
		data["discordUsername"] = app.discord.username
		data["discordRequired"] = app.config.Section("discord").Key("required").MustBool(false)
		data["discordSendPINMessage"] = template.HTML(app.storage.lang.User[lang].Strings.template("sendPINDiscord", tmpl{
			"command":        `<span class="text-black dark:text-white font-mono">/` + app.config.Section("discord").Key("start_command").MustString("start") + `</span>`,
			"server_channel": app.discord.serverChannelName,
		}))
		data["discordServerName"] = app.discord.serverName
		data["discordInviteLink"] = app.discord.InviteChannel.Name != ""
	}
	if data["linkResetEnabled"].(bool) {
		data["resetPasswordUsername"] = app.config.Section("user_page").Key("allow_pwr_username").MustBool(true)
		data["resetPasswordEmail"] = app.config.Section("user_page").Key("allow_pwr_email").MustBool(true)
		data["resetPasswordContactMethod"] = app.config.Section("user_page").Key("allow_pwr_contact_method").MustBool(true)
	}

	pageMessagesExist := map[string]bool{}
	pageMessages := map[string]CustomContent{}
	pageMessages["Login"], pageMessagesExist["Login"] = app.storage.GetCustomContentKey("UserLogin")
	pageMessages["Page"], pageMessagesExist["Page"] = app.storage.GetCustomContentKey("UserPage")

	for name, msg := range pageMessages {
		if !pageMessagesExist[name] {
			continue
		}
		data[name+"MessageEnabled"] = msg.Enabled
		if !msg.Enabled {
			continue
		}
		// We don't template here, since the username is only known after login.
		data[name+"MessageContent"] = template.HTML(markdown.ToHTML([]byte(msg.Content), nil, markdownRenderer))
	}

	app.gcHTML(gc, http.StatusOK, "user.html", UserPage, data)
}

func (app *appContext) ResetPassword(gc *gin.Context) {
	isBot := strings.Contains(gc.Request.Header.Get("User-Agent"), "Bot")
	setPassword := app.config.Section("password_resets").Key("set_password").MustBool(false)
	pin := gc.Query("pin")
	if pin == "" {
		app.NoRouteHandler(gc)
		return
	}
	app.pushResources(gc, PWRPage)
	lang := app.getLang(gc, PWRPage, app.storage.lang.chosenPWRLang)
	data := gin.H{
		"contactMessage":    app.config.Section("ui").Key("contact_message").String(),
		"strings":           app.storage.lang.PasswordReset[lang].Strings,
		"success":           false,
		"customSuccessCard": false,
	}
	pwr, isInternal := app.internalPWRs[pin]
	// if isInternal && setPassword {
	if setPassword {
		data["helpMessage"] = app.config.Section("ui").Key("help_message").String()
		data["successMessage"] = app.config.Section("ui").Key("success_message").String()
		data["jfLink"] = app.config.Section("ui").Key("redirect_url").String()
		data["redirectToJellyfin"] = app.config.Section("ui").Key("auto_redirect").MustBool(false)
		data["validate"] = app.config.Section("password_validation").Key("enabled").MustBool(false)
		data["requirements"] = app.validator.getCriteria()
		data["strings"] = app.storage.lang.PasswordReset[lang].Strings
		data["validationStrings"] = app.storage.lang.User[lang].validationStringsJSON
		// ewwwww, reusing an existing field, FIXME!
		data["notifications"] = app.storage.lang.User[lang].notificationsJSON
		data["langName"] = lang
		data["passwordReset"] = true
		data["telegramEnabled"] = false
		data["discordEnabled"] = false
		data["matrixEnabled"] = false
		data["captcha"] = app.config.Section("captcha").Key("enabled").MustBool(false)
		data["reCAPTCHA"] = app.config.Section("captcha").Key("recaptcha").MustBool(false)
		data["reCAPTCHASiteKey"] = app.config.Section("captcha").Key("recaptcha_site_key").MustString("")
		data["pwrPIN"] = pin
		app.gcHTML(gc, http.StatusOK, "form-loader.html", PWRPage, data)
		return
	}
	defer app.gcHTML(gc, http.StatusOK, "password-reset.html", PWRPage, data)
	// If it's a bot, pretend to be a success so the preview is nice.
	if isBot {
		app.debug.Println(lm.IgnoreBotPWR)
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
	var err error
	var username string
	if !isInternal && !setPassword {
		resp, err = app.jf.ResetPassword(pin)
	} else if time.Now().After(pwr.Expiry) {
		app.debug.Printf(lm.FailedChangePassword, lm.Jellyfin, "?", fmt.Sprintf(lm.ExpiredPIN, pin))
		app.NoRouteHandler(gc)
		return
	} else {
		err = app.jf.ResetPasswordAdmin(pwr.ID)
		if err != nil {
			app.err.Printf(lm.FailedChangePassword, lm.Jellyfin, "?", err)
		} else {
			err = app.jf.SetPassword(pwr.ID, "", pin)
		}
		username = pwr.Username
	}

	if err == nil && (isInternal || resp.Success) {
		data["success"] = true
		data["pin"] = pin
		if !isInternal {
			username = resp.UsersReset[0]
		}
	} else {
		app.err.Printf(lm.FailedChangePassword, lm.Jellyfin, "?", err)
	}

	// Only log PWRs we know the user for.
	if username != "" {
		jfUser, err := app.jf.UserByName(username, false)
		if err == nil {
			app.storage.SetActivityKey(shortuuid.New(), Activity{
				Type:       ActivityResetPassword,
				UserID:     jfUser.ID,
				SourceType: ActivityUser,
				Source:     jfUser.ID,
				Time:       time.Now(),
			}, gc, true)
		}
	}

	if app.config.Section("ombi").Key("enabled").MustBool(false) {
		jfUser, err := app.jf.UserByName(username, false)
		if err != nil {
			app.err.Printf(lm.FailedGetUser, username, lm.Jellyfin, err)
			return
		}
		ombiUser, err := app.getOmbiUser(jfUser.ID)
		if err != nil {
			app.err.Printf(lm.FailedGetUser, username, lm.Ombi, err)
			return
		}
		ombiUser["password"] = pin
		err = app.ombi.ModifyUser(ombiUser)
		if err != nil {
			app.err.Printf(lm.FailedChangePassword, lm.Ombi, ombiUser["userName"], err)
			return
		}
		app.debug.Printf(lm.ChangePassword, lm.Ombi, ombiUser["userName"])
	}
}

// @Summary returns the captcha image corresponding to the given ID.
// @Param code path string true "invite code"
// @Param captchaID path string true "captcha ID"
// @Tags Other
// @Router /captcha/img/{code}/{captchaID} [get]
func (app *appContext) GetCaptcha(gc *gin.Context) {
	code := gc.Param("invCode")
	isPWR := gc.Query("pwr") == "true"
	captchaID := gc.Param("captchaID")
	var inv Invite
	var capt Captcha
	ok := true
	if !isPWR {
		inv, ok = app.storage.GetInvitesKey(code)
		if !ok {
			app.gcHTML(gc, 404, "invalidCode.html", OtherPage, gin.H{
				"contactMessage": app.config.Section("ui").Key("contact_message").String(),
			})
		}
		if inv.Captchas != nil {
			capt, ok = inv.Captchas[captchaID]
		} else {
			ok = false
		}
	} else {
		capt, ok = app.pwrCaptchas[code]
	}
	if !ok {
		respondBool(400, false, gc)
		return
	}
	gc.Data(200, "image/png", capt.Image)
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
	isPWR := gc.Query("pwr") == "true"
	var inv Invite
	ok := true
	if !isPWR {
		inv, ok = app.storage.GetInvitesKey(code)
	}

	if !ok {
		app.gcHTML(gc, 404, "invalidCode.html", OtherPage, gin.H{
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		})
	}
	capt, err := captcha.New(300, 100)
	if err != nil {
		app.err.Printf(lm.FailedGenerateCaptcha, err)
		respondBool(500, false, gc)
		return
	}
	if !isPWR && inv.Captchas == nil {
		inv.Captchas = map[string]Captcha{}
	}
	captchaID := genAuthToken()
	var buf bytes.Buffer
	if err := capt.WriteImage(bufio.NewWriter(&buf)); err != nil {
		app.err.Printf(lm.FailedGenerateCaptcha, err)
		respondBool(500, false, gc)
		return
	}
	if isPWR {
		if app.pwrCaptchas == nil {
			app.pwrCaptchas = map[string]Captcha{}
		}
		app.pwrCaptchas[code] = Captcha{
			Answer:    capt.Text,
			Image:     buf.Bytes(),
			Generated: time.Now(),
		}
	} else {
		inv.Captchas[captchaID] = Captcha{
			Answer:    capt.Text,
			Image:     buf.Bytes(),
			Generated: time.Now(),
		}
		app.storage.SetInvitesKey(code, inv)
	}
	gc.JSON(200, genCaptchaDTO{captchaID})
	return
}

func (app *appContext) verifyCaptcha(code, id, text string, isPWR bool) bool {
	reCAPTCHA := app.config.Section("captcha").Key("recaptcha").MustBool(false)
	if !reCAPTCHA {
		// internal CAPTCHA
		var c Captcha
		ok := true
		if !isPWR {
			inv, ok := app.storage.GetInvitesKey(code)
			if !ok {
				app.debug.Printf(lm.InvalidInviteCode, code)
				return false
			}
			if !isPWR && inv.Captchas == nil {
				app.debug.Printf(lm.CaptchaNotFound, id, code)
				return false
			}
			c, ok = inv.Captchas[id]
		} else {
			c, ok = app.pwrCaptchas[code]
		}
		if !ok {
			app.debug.Printf(lm.CaptchaNotFound, id, code)
			return false
		}
		return strings.ToLower(c.Answer) == strings.ToLower(text)
	}

	// reCAPTCHA

	msg := ReCaptchaRequestDTO{
		Secret:   app.config.Section("captcha").Key("recaptcha_secret_key").MustString(""),
		Response: text,
	}
	// Why doesn't this endpoint accept JSON???
	urlencode := url.Values{}
	urlencode.Set("secret", msg.Secret)
	urlencode.Set("response", msg.Response)

	req, _ := http.NewRequest("POST", "https://www.google.com/recaptcha/api/siteverify", strings.NewReader(urlencode.Encode()))

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	err = common.GenericErr(resp.StatusCode, err)
	if err != nil {
		app.err.Printf(lm.FailedVerifyReCAPTCHA, err)
		return false
	}
	defer resp.Body.Close()
	var data ReCaptchaResponseDTO
	body, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &data)
	if err != nil {
		app.err.Printf(lm.FailedVerifyReCAPTCHA, err)
		return false
	}

	hostname := app.config.Section("captcha").Key("recaptcha_hostname").MustString("")
	if strings.ToLower(data.Hostname) != strings.ToLower(hostname) && data.Hostname != "" {
		err = fmt.Errorf(lm.InvalidHostname, hostname, data.Hostname)
		app.err.Printf(lm.FailedVerifyReCAPTCHA, err)
		return false
	}

	if len(data.ErrorCodes) > 0 {
		app.err.Printf(lm.AdditionalErrors, lm.ReCAPTCHA, data.ErrorCodes)
		return false
	}

	return data.Success
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
	isPWR := gc.Query("pwr") == "true"
	captchaID := gc.Param("captchaID")
	text := gc.Param("text")
	var inv Invite
	var capt Captcha
	var ok bool
	if !isPWR {
		inv, ok = app.storage.GetInvitesKey(code)
		if !ok {
			app.gcHTML(gc, 404, "invalidCode.html", OtherPage, gin.H{
				"contactMessage": app.config.Section("ui").Key("contact_message").String(),
			})
			return
		}
		if inv.Captchas != nil {
			capt, ok = inv.Captchas[captchaID]
		} else {
			ok = false
		}
	} else {
		capt, ok = app.pwrCaptchas[code]
	}
	if !ok {
		respondBool(400, false, gc)
		return
	}
	if strings.ToLower(capt.Answer) != strings.ToLower(text) {
		respondBool(400, false, gc)
		return
	}
	respondBool(204, true, gc)
	return
}

func (app *appContext) NewUserFromConfirmationKey(invite Invite, key string, lang string, gc *gin.Context) {
	fail := func() {
		app.gcHTML(gc, 404, "404.html", OtherPage, gin.H{
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		})
	}
	var req ConfirmationKey
	if app.ConfirmationKeys == nil {
		fail()
		return
	}

	invKeys, ok := app.ConfirmationKeys[invite.Code]
	if !ok {
		fail()
		return
	}
	req, ok = invKeys[key]
	if !ok {
		fail()
		return
	}
	token, err := jwt.Parse(key, checkToken)
	if err != nil {
		fail()
		app.debug.Printf(lm.FailedParseJWT, err)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	expiry := time.Unix(int64(claims["exp"].(float64)), 0)
	if !(ok && token.Valid && claims["invite"].(string) == invite.Code && claims["type"].(string) == "confirmation" && expiry.After(time.Now())) {
		fail()
		app.debug.Printf(lm.InvalidJWT)
		return
	}

	sourceType, source := invite.Source()

	var profile *Profile = nil
	if invite.Profile != "" {
		p, ok := app.storage.GetProfileKey(invite.Profile)
		if !ok {
			app.debug.Printf(lm.FailedGetProfile+lm.FallbackToDefault, invite.Profile)
			p = app.storage.GetDefaultProfile()
		}
		profile = &p
	}

	// FIXME: Email and contract method linking?????

	nu /*wg*/, _ := app.NewUserPostVerification(NewUserParams{
		Req:                 req.newUserDTO,
		SourceType:          sourceType,
		Source:              source,
		ContextForIPLogging: gc,
		Profile:             profile,
	})
	if !nu.Success {
		nu.Log()
	}
	if !nu.Created {
		// respond(nu.Status, nu.Message, gc)
		fail()
		return
	}
	app.checkInvite(req.Code, true, req.Username)

	app.PostNewUserFromInvite(nu, req, profile, invite)

	jfLink := app.config.Section("ui").Key("redirect_url").String()
	if app.config.Section("ui").Key("auto_redirect").MustBool(false) {
		gc.Redirect(301, jfLink)
	} else {
		app.gcHTML(gc, http.StatusOK, "create-success.html", OtherPage, gin.H{
			"strings":        app.storage.lang.User[lang].Strings,
			"successMessage": app.config.Section("ui").Key("success_message").String(),
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
			"jfLink":         jfLink,
		})
	}
	app.confirmationKeysLock.Lock()
	// Re-fetch invKeys just incase an update occurred
	invKeys, ok = app.ConfirmationKeys[invite.Code]
	if !ok {
		fail()
		return
	}
	delete(invKeys, key)
	app.ConfirmationKeys[invite.Code] = invKeys
	app.confirmationKeysLock.Unlock()

	// These don't need to complete anytime soon
	// wg.Wait()
	return
}

func (app *appContext) InviteProxy(gc *gin.Context) {
	app.pushResources(gc, FormPage)
	lang := app.getLang(gc, FormPage, app.storage.lang.chosenUserLang)

	/* Don't actually check if the invite is valid, just if it exists, just so the page loads quicker. Invite is actually checked on submit anyway. */
	// if app.checkInvite(code, false, "") {
	invite, ok := app.storage.GetInvitesKey(gc.Param("invCode"))
	if !ok {
		app.gcHTML(gc, 404, "invalidCode.html", FormPage, gin.H{
			"contactMessage": app.config.Section("ui").Key("contact_message").String(),
		})
		return
	}

	if key := gc.Query("key"); key != "" && app.config.Section("email_confirmation").Key("enabled").MustBool(false) {
		app.NewUserFromConfirmationKey(invite, key, lang, gc)
		return
	}

	email := invite.SendTo
	if strings.Contains(email, "Failed") || !strings.Contains(email, "@") {
		email = ""
	}
	telegram := telegramEnabled && app.config.Section("telegram").Key("show_on_reg").MustBool(true)
	discord := discordEnabled && app.config.Section("discord").Key("show_on_reg").MustBool(true)
	matrix := matrixEnabled && app.config.Section("matrix").Key("show_on_reg").MustBool(true)

	userPageAddress := app.ExternalURI(gc) + PAGES.MyAccount

	fromUser := ""
	if invite.ReferrerJellyfinID != "" {
		sender, err := app.jf.UserByID(invite.ReferrerJellyfinID, false)
		if err == nil {
			fromUser = sender.Name
		}
	}

	data := gin.H{
		"contactMessage":     app.config.Section("ui").Key("contact_message").String(),
		"helpMessage":        app.config.Section("ui").Key("help_message").String(),
		"successMessage":     app.config.Section("ui").Key("success_message").String(),
		"jfLink":             app.config.Section("ui").Key("redirect_url").String(),
		"redirectToJellyfin": app.config.Section("ui").Key("auto_redirect").MustBool(false),
		"validate":           app.config.Section("password_validation").Key("enabled").MustBool(false),
		"requirements":       app.validator.getCriteria(),
		"email":              email,
		"username":           !app.config.Section("email").Key("no_username").MustBool(false),
		"strings":            app.storage.lang.User[lang].Strings,
		"validationStrings":  app.storage.lang.User[lang].validationStringsJSON,
		// ewwwww, reusing an existing field, FIXME!
		"notifications":     app.storage.lang.User[lang].notificationsJSON,
		"code":              invite.Code,
		"confirmation":      app.config.Section("email_confirmation").Key("enabled").MustBool(false),
		"userExpiry":        invite.UserExpiry,
		"userExpiryMonths":  invite.UserMonths,
		"userExpiryDays":    invite.UserDays,
		"userExpiryHours":   invite.UserHours,
		"userExpiryMinutes": invite.UserMinutes,
		"userExpiryMessage": app.storage.lang.User[lang].Strings.get("yourAccountIsValidUntil"),
		"langName":          lang,
		"passwordReset":     false,
		"customSuccessCard": false,
		"telegramEnabled":   telegram,
		"discordEnabled":    discord,
		"matrixEnabled":     matrix,
		"emailRequired":     app.config.Section("email").Key("required").MustBool(false),
		"captcha":           app.config.Section("captcha").Key("enabled").MustBool(false),
		"reCAPTCHA":         app.config.Section("captcha").Key("recaptcha").MustBool(false),
		"reCAPTCHASiteKey":  app.config.Section("captcha").Key("recaptcha_site_key").MustString(""),
		"userPageEnabled":   app.config.Section("user_page").Key("enabled").MustBool(false),
		"userPageAddress":   userPageAddress,
		"fromUser":          fromUser,
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
		data["discordSendPINMessage"] = template.HTML(app.storage.lang.User[lang].Strings.template("sendPINDiscord", tmpl{
			"command":        `<span class="text-black dark:text-white font-mono">/` + app.config.Section("discord").Key("start_command").MustString("start") + `</span>`,
			"server_channel": app.discord.serverChannelName,
		}))
		data["discordServerName"] = app.discord.serverName
		data["discordInviteLink"] = app.discord.InviteChannel.Name != ""
	}
	if msg, ok := app.storage.GetCustomContentKey("PostSignupCard"); ok && msg.Enabled {
		data["customSuccessCard"] = true
		// We don't template here, since the username is only known after login.
		data["customSuccessCardContent"] = template.HTML(markdown.ToHTML(
			[]byte(templateEmail(
				msg.Content,
				msg.Variables,
				msg.Conditionals,
				map[string]interface{}{
					"username":     "{username}",
					"myAccountURL": userPageAddress,
				},
			),
			), nil, markdownRenderer,
		))
	}

	// if discordEnabled {
	// 	pin := ""
	// 	for _, token := range app.discord.tokens {
	// 		if
	app.gcHTML(gc, http.StatusOK, "form-loader.html", OtherPage, data)
}

func (app *appContext) NoRouteHandler(gc *gin.Context) {
	app.pushResources(gc, OtherPage)
	app.gcHTML(gc, 404, "404.html", OtherPage, gin.H{
		"contactMessage": app.config.Section("ui").Key("contact_message").String(),
	})
}
