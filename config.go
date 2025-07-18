package main

import (
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/common"
	"github.com/hrfee/jfa-go/easyproxy"
	lm "github.com/hrfee/jfa-go/logmessages"
	"gopkg.in/ini.v1"
)

var emailEnabled = false
var messagesEnabled = false
var telegramEnabled = false
var discordEnabled = false
var matrixEnabled = false

// URL subpaths. Ignore the "Current" field.
var PAGES = PagePaths{}

func (app *appContext) GetPath(sect, key string) (fs.FS, string) {
	val := app.config.Section(sect).Key(key).MustString("")
	if strings.HasPrefix(val, "jfa-go:") {
		return localFS, strings.TrimPrefix(val, "jfa-go:")
	}
	dir, file := filepath.Split(val)
	return os.DirFS(dir), file
}

func (app *appContext) MustSetValue(section, key, val string) {
	app.config.Section(section).Key(key).SetValue(app.config.Section(section).Key(key).MustString(val))
}

func (app *appContext) MustSetURLPath(section, key, val string) {
	if !strings.HasPrefix(val, "/") && val != "" {
		val = "/" + val
	}
	app.MustSetValue(section, key, val)
}

func FixFullURL(v string) string {
	// Keep relative paths relative
	if strings.HasPrefix(v, "/") {
		return v
	}
	if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
		v = "http://" + v
	}
	return v
}

func FormatSubpath(path string) string {
	if path == "/" {
		return ""
	}
	return strings.TrimSuffix(path, "/")
}

func (app *appContext) MustCorrectURL(section, key, value string) {
	v := app.config.Section(section).Key(key).String()
	if v == "" {
		v = value
	}
	v = FixFullURL(v)
	app.config.Section(section).Key(key).SetValue(v)
}

// ExternalDomain returns the Host for the request, using the fixed app.externalDomain value unless app.UseProxyHost is true.
func (app *appContext) ExternalDomain(gc *gin.Context) string {
	if !app.UseProxyHost || gc.Request.Host == "" {
		return app.externalDomain
	}
	return gc.Request.Host
}

// ExternalDomainNoPort attempts to return app.ExternalDomain() with the port removed. If the internally-used method fails, it is assumed the domain has no port anyway.
func (app *appContext) ExternalDomainNoPort(gc *gin.Context) string {
	domain := app.ExternalDomain(gc)
	host, _, err := net.SplitHostPort(domain)
	if err != nil {
		return domain
	}
	return host
}

// ExternalURI returns the External URI of jfa-go's root directory (by default, where the admin page is), using the fixed app.externalURI value unless app.UseProxyHost is true and gc is not nil.
// When nil is passed, app.externalURI is returned.
func (app *appContext) ExternalURI(gc *gin.Context) string {
	if gc == nil {
		return app.externalURI
	}

	var proto string
	if gc.Request.TLS != nil || gc.Request.Header.Get("X-Forwarded-Proto") == "https" || gc.Request.Header.Get("X-Forwarded-Protocol") == "https" {
		proto = "https://"
	} else {
		proto = "http://"
	}

	// app.debug.Printf("Request: %+v\n", gc.Request)
	if app.UseProxyHost && gc.Request.Host != "" {
		return proto + gc.Request.Host + PAGES.Base
	}
	return app.externalURI
}

func (app *appContext) EvaluateRelativePath(gc *gin.Context, path string) string {
	if !strings.HasPrefix(path, "/") {
		return path
	}

	var proto string
	if gc.Request.TLS != nil || gc.Request.Header.Get("X-Forwarded-Proto") == "https" || gc.Request.Header.Get("X-Forwarded-Protocol") == "https" {
		proto = "https://"
	} else {
		proto = "http://"
	}

	return proto + app.ExternalDomain(gc) + path
}

func (app *appContext) loadConfig() error {
	var err error
	app.config, err = ini.ShadowLoad(app.configPath)
	if err != nil {
		return err
	}

	// URLs
	app.MustSetURLPath("ui", "url_base", "")
	app.MustSetURLPath("url_paths", "admin", "")
	app.MustSetURLPath("url_paths", "user_page", "/my/account")
	app.MustSetURLPath("url_paths", "form", "/invite")
	PAGES.Base = FormatSubpath(app.config.Section("ui").Key("url_base").String())
	PAGES.Admin = FormatSubpath(app.config.Section("url_paths").Key("admin").String())
	PAGES.MyAccount = FormatSubpath(app.config.Section("url_paths").Key("user_page").String())
	PAGES.Form = FormatSubpath(app.config.Section("url_paths").Key("form").String())
	if !(app.config.Section("user_page").Key("enabled").MustBool(true)) {
		PAGES.MyAccount = "disabled"
	}
	if PAGES.Base == PAGES.Form || PAGES.Base == "/accounts" || PAGES.Base == "/settings" || PAGES.Base == "/activity" {
		app.err.Printf(lm.BadURLBase, PAGES.Base)
	}
	app.info.Printf(lm.SubpathBlockMessage, PAGES.Base, PAGES.Admin, PAGES.MyAccount, PAGES.Form)

	app.MustCorrectURL("jellyfin", "server", "")
	app.MustCorrectURL("jellyfin", "public_server", app.config.Section("jellyfin").Key("server").String())
	app.MustCorrectURL("ui", "redirect_url", app.config.Section("jellyfin").Key("public_server").String())

	for _, key := range app.config.Section("files").Keys() {
		if name := key.Name(); name != "html_templates" && name != "lang_files" {
			key.SetValue(key.MustString(filepath.Join(app.dataPath, (key.Name() + ".json"))))
		}
	}
	for _, key := range []string{"user_configuration", "user_displayprefs", "user_profiles", "ombi_template", "invites", "emails", "user_template", "custom_emails", "users", "telegram_users", "discord_users", "matrix_users", "announcements", "custom_user_page_content"} {
		app.config.Section("files").Key(key).SetValue(app.config.Section("files").Key(key).MustString(filepath.Join(app.dataPath, (key + ".json"))))
	}
	for _, key := range []string{"matrix_sql"} {
		app.config.Section("files").Key(key).SetValue(app.config.Section("files").Key(key).MustString(filepath.Join(app.dataPath, (key + ".db"))))
	}

	// If true, app.ExternalDomain() will return one based on the reported Host (ideally reported in "Host" or "X-Forwarded-Host" by the reverse proxy), falling back to app.externalDomain if not set.
	app.UseProxyHost = app.config.Section("ui").Key("use_proxy_host").MustBool(false)
	app.externalURI = strings.TrimSuffix(strings.TrimSuffix(app.config.Section("ui").Key("jfa_url").MustString(""), "/invite"), "/")
	if !strings.HasSuffix(app.externalURI, PAGES.Base) {
		app.err.Println(lm.NoURLSuffix)
	}
	if app.externalURI == "" {
		if app.UseProxyHost {
			app.err.Println(lm.NoExternalHost + lm.LoginWontSave + lm.SetExternalHostDespiteUseProxyHost)
		} else {
			app.err.Println(lm.NoExternalHost + lm.LoginWontSave)
		}
	}
	u, err := url.Parse(app.externalURI)
	if err == nil {
		app.externalDomain = u.Hostname()
	}

	app.config.Section("email").Key("no_username").SetValue(strconv.FormatBool(app.config.Section("email").Key("no_username").MustBool(false)))

	app.MustSetValue("password_resets", "email_html", "jfa-go:"+"email.html")
	app.MustSetValue("password_resets", "email_text", "jfa-go:"+"email.txt")

	app.MustSetValue("invite_emails", "email_html", "jfa-go:"+"invite-email.html")
	app.MustSetValue("invite_emails", "email_text", "jfa-go:"+"invite-email.txt")

	app.MustSetValue("email_confirmation", "email_html", "jfa-go:"+"confirmation.html")
	app.MustSetValue("email_confirmation", "email_text", "jfa-go:"+"confirmation.txt")

	app.MustSetValue("notifications", "expiry_html", "jfa-go:"+"expired.html")
	app.MustSetValue("notifications", "expiry_text", "jfa-go:"+"expired.txt")

	app.MustSetValue("notifications", "created_html", "jfa-go:"+"created.html")
	app.MustSetValue("notifications", "created_text", "jfa-go:"+"created.txt")

	app.MustSetValue("deletion", "email_html", "jfa-go:"+"deleted.html")
	app.MustSetValue("deletion", "email_text", "jfa-go:"+"deleted.txt")

	app.MustSetValue("smtp", "hello_hostname", "localhost")
	app.MustSetValue("smtp", "cert_validation", "true")
	app.MustSetValue("smtp", "auth_type", "4")
	app.MustSetValue("smtp", "port", "465")

	app.MustSetValue("activity_log", "keep_n_records", "1000")
	app.MustSetValue("activity_log", "delete_after_days", "90")

	sc := app.config.Section("discord").Key("start_command").MustString("start")
	app.config.Section("discord").Key("start_command").SetValue(strings.TrimPrefix(strings.TrimPrefix(sc, "/"), "!"))

	// Deletion template is good enough for these as well.
	app.MustSetValue("disable_enable", "disabled_html", "jfa-go:"+"deleted.html")
	app.MustSetValue("disable_enable", "disabled_text", "jfa-go:"+"deleted.txt")
	app.MustSetValue("disable_enable", "enabled_html", "jfa-go:"+"deleted.html")
	app.MustSetValue("disable_enable", "enabled_text", "jfa-go:"+"deleted.txt")

	app.MustSetValue("welcome_email", "email_html", "jfa-go:"+"welcome.html")
	app.MustSetValue("welcome_email", "email_text", "jfa-go:"+"welcome.txt")

	app.MustSetValue("template_email", "email_html", "jfa-go:"+"template.html")
	app.MustSetValue("template_email", "email_text", "jfa-go:"+"template.txt")

	app.MustSetValue("user_expiry", "behaviour", "disable_user")
	app.MustSetValue("user_expiry", "email_html", "jfa-go:"+"user-expired.html")
	app.MustSetValue("user_expiry", "email_text", "jfa-go:"+"user-expired.txt")

	app.MustSetValue("user_expiry", "adjustment_email_html", "jfa-go:"+"expiry-adjusted.html")
	app.MustSetValue("user_expiry", "adjustment_email_text", "jfa-go:"+"expiry-adjusted.txt")

	app.MustSetValue("email", "collect", "true")

	app.MustSetValue("matrix", "topic", "Jellyfin notifications")
	app.MustSetValue("matrix", "show_on_reg", "true")

	app.MustSetValue("discord", "show_on_reg", "true")

	app.MustSetValue("telegram", "show_on_reg", "true")

	app.MustSetValue("backups", "every_n_minutes", "1440")
	app.MustSetValue("backups", "path", filepath.Join(app.dataPath, "backups"))
	app.MustSetValue("backups", "keep_n_backups", "20")
	app.MustSetValue("backups", "keep_previous_version_backup", "true")

	app.config.Section("jellyfin").Key("version").SetValue(version)
	app.config.Section("jellyfin").Key("device").SetValue("jfa-go")
	app.config.Section("jellyfin").Key("device_id").SetValue(fmt.Sprintf("jfa-go-%s-%s", version, commit))

	app.MustSetValue("jellyfin", "cache_timeout", "30")
	app.MustSetValue("jellyfin", "web_cache_async_timeout", "1")
	app.MustSetValue("jellyfin", "web_cache_sync_timeout", "10")

	LOGIP = app.config.Section("advanced").Key("log_ips").MustBool(false)
	LOGIPU = app.config.Section("advanced").Key("log_ips_users").MustBool(false)

	app.MustSetValue("advanced", "auth_retry_count", "6")
	app.MustSetValue("advanced", "auth_retry_gap", "10")

	app.MustSetValue("ui", "port", "8056")
	app.MustSetValue("advanced", "tls_port", "8057")

	app.MustSetValue("advanced", "value_log_size", "512")

	pwrMethods := []string{"allow_pwr_username", "allow_pwr_email", "allow_pwr_contact_method"}
	allDisabled := true
	for _, v := range pwrMethods {
		if app.config.Section("user_page").Key(v).MustBool(true) {
			allDisabled = false
		}
	}
	if allDisabled {
		app.info.Println(lm.EnableAllPWRMethods)
		for _, v := range pwrMethods {
			app.config.Section("user_page").Key(v).SetValue("true")
		}
	}

	messagesEnabled = app.config.Section("messages").Key("enabled").MustBool(false)
	telegramEnabled = app.config.Section("telegram").Key("enabled").MustBool(false)
	discordEnabled = app.config.Section("discord").Key("enabled").MustBool(false)
	matrixEnabled = app.config.Section("matrix").Key("enabled").MustBool(false)
	if !messagesEnabled {
		emailEnabled = false
		telegramEnabled = false
		discordEnabled = false
		matrixEnabled = false
	} else if app.config.Section("email").Key("method").MustString("") == "" {
		emailEnabled = false
	} else {
		emailEnabled = true
	}
	if !emailEnabled && !telegramEnabled && !discordEnabled && !matrixEnabled {
		messagesEnabled = false
	}

	if app.proxyEnabled = app.config.Section("advanced").Key("proxy").MustBool(false); app.proxyEnabled {
		app.proxyConfig = easyproxy.ProxyConfig{}
		app.proxyConfig.Protocol = easyproxy.HTTP
		if strings.Contains(app.config.Section("advanced").Key("proxy_protocol").MustString("http"), "socks") {
			app.proxyConfig.Protocol = easyproxy.SOCKS5
		}
		app.proxyConfig.Addr = app.config.Section("advanced").Key("proxy_address").MustString("")
		app.proxyConfig.User = app.config.Section("advanced").Key("proxy_user").MustString("")
		app.proxyConfig.Password = app.config.Section("advanced").Key("proxy_password").MustString("")
		app.proxyTransport, err = easyproxy.NewTransport(app.proxyConfig)
		if err != nil {
			app.err.Printf(lm.FailedInitProxy, app.proxyConfig.Addr, err)
			// As explained in lm.FailedInitProxy, sleep here might grab the admin's attention,
			// Since we don't crash on this failing.
			time.Sleep(15 * time.Second)
			app.proxyEnabled = false
		} else {
			app.proxyEnabled = true
			app.info.Printf(lm.InitProxy, app.proxyConfig.Addr)
		}
	}

	app.MustSetValue("updates", "enabled", "true")
	releaseChannel := app.config.Section("updates").Key("channel").String()
	if app.config.Section("updates").Key("enabled").MustBool(false) {
		v := version
		if releaseChannel == "stable" {
			if version == "git" {
				v = "0.0.0"
			}
		} else if releaseChannel == "unstable" {
			v = "git"
		}
		app.updater = newUpdater(baseURL, namespace, repo, v, commit, updater)
		if app.proxyEnabled {
			app.updater.SetTransport(app.proxyTransport)
		}
	}
	if releaseChannel == "" {
		if version == "git" {
			releaseChannel = "unstable"
		} else {
			releaseChannel = "stable"
		}
		app.MustSetValue("updates", "channel", releaseChannel)
	}

	substituteStrings = app.config.Section("jellyfin").Key("substitute_jellyfin_strings").MustString("")

	if substituteStrings != "" {
		v := app.config.Section("ui").Key("success_message")
		v.SetValue(strings.ReplaceAll(v.String(), "Jellyfin", substituteStrings))
	}

	oldFormLang := app.config.Section("ui").Key("language").MustString("")
	if oldFormLang != "" {
		app.storage.lang.chosenUserLang = oldFormLang
	}
	newFormLang := app.config.Section("ui").Key("language-form").MustString("")
	if newFormLang != "" {
		app.storage.lang.chosenUserLang = newFormLang
	}
	app.storage.lang.chosenAdminLang = app.config.Section("ui").Key("language-admin").MustString("en-us")
	app.storage.lang.chosenEmailLang = app.config.Section("email").Key("language").MustString("en-us")
	app.storage.lang.chosenPWRLang = app.config.Section("password_resets").Key("language").MustString("en-us")
	app.storage.lang.chosenTelegramLang = app.config.Section("telegram").Key("language").MustString("en-us")

	app.email = NewEmailer(app)

	return nil
}

func (app *appContext) PatchConfigBase() {
	conf := app.configBase
	// Load language options
	formOptions := app.storage.lang.User.getOptions()
	pwrOptions := app.storage.lang.PasswordReset.getOptions()
	adminOptions := app.storage.lang.Admin.getOptions()
	emailOptions := app.storage.lang.Email.getOptions()
	telegramOptions := app.storage.lang.Email.getOptions()

	for i, section := range app.configBase.Sections {
		if section.Section == "updates" && updater == "" {
			section.Meta.Disabled = true
		}
		for j, setting := range section.Settings {
			if section.Section == "ui" {
				if setting.Setting == "language-form" {
					setting.Options = formOptions
					setting.Value = "en-us"
				} else if setting.Setting == "language-admin" {
					setting.Options = adminOptions
					setting.Value = "en-us"
				}
			} else if section.Section == "password_resets" {
				if setting.Setting == "language" {
					setting.Options = pwrOptions
					setting.Value = "en-us"
				}
			} else if section.Section == "email" {
				if setting.Setting == "language" {
					setting.Options = emailOptions
					setting.Value = "en-us"
				}
			} else if section.Section == "telegram" {
				if setting.Setting == "language" {
					setting.Options = telegramOptions
					setting.Value = "en-us"
				}
			} else if section.Section == "smtp" {
				if setting.Setting == "ssl_cert" && PLATFORM == "windows" {
					// Not accurate but the effect is hiding the option, which we want.
					setting.Deprecated = true
				}
			} else if section.Section == "matrix" {
				if setting.Setting == "encryption" && !MatrixE2EE() {
					// Not accurate but the effect is hiding the option, which we want.
					setting.Deprecated = true
				}
			}
			val := app.config.Section(section.Section).Key(setting.Setting)
			switch setting.Type {
			case "list":
				setting.Value = val.StringsWithShadows("|")
			case "text", "email", "select", "password", "note":
				setting.Value = val.MustString("")
			case "number":
				setting.Value = val.MustInt(0)
			case "bool":
				setting.Value = val.MustBool(false)
			}
			section.Settings[j] = setting
		}
		conf.Sections[i] = section
	}
	app.patchedConfig = conf
}

func (app *appContext) PatchConfigDiscordRoles() {
	if !discordEnabled {
		return
	}
	r, err := app.discord.ListRoles()
	if err != nil {
		return
	}
	roles := make([]common.Option, len(r)+1)
	roles[0] = common.Option{"", "None"}
	for i, role := range r {
		roles[i+1] = role
	}

	for i, section := range app.patchedConfig.Sections {
		if section.Section != "discord" {
			continue
		}
		for j, setting := range section.Settings {
			if setting.Setting != "apply_role" {
				continue
			}
			setting.Options = roles
			section.Settings[j] = setting
		}
		app.patchedConfig.Sections[i] = section
	}
}
