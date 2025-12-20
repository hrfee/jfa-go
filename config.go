package main

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
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

type Config struct {
	*ini.File
	proxyTransport *http.Transport
	proxyConfig    *easyproxy.ProxyConfig
}

var emailEnabled = false
var messagesEnabled = false
var telegramEnabled = false
var discordEnabled = false
var matrixEnabled = false

// URL subpaths. Ignore the "Current" field, it's populated when in copies of the struct used for page templating.
// IMPORTANT: When linking straight to a page, rather than appending further to the URL (like accessing an API route), append a /.
var PAGES = PagePaths{}

func (config *Config) GetPath(sect, key string) (fs.FS, string) {
	val := config.Section(sect).Key(key).MustString("")
	if strings.HasPrefix(val, "jfa-go:") {
		return localFS, strings.TrimPrefix(val, "jfa-go:")
	}
	dir, file := filepath.Split(val)
	return os.DirFS(dir), file
}

func (config *Config) MustSetValue(section, key, val string) {
	config.Section(section).Key(key).SetValue(config.Section(section).Key(key).MustString(val))
}

func (config *Config) MustSetURLPath(section, key, val string) {
	if !strings.HasPrefix(val, "/") && val != "" {
		val = "/" + val
	}
	config.MustSetValue(section, key, val)
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

func MustGetNonEmptyURL(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func FormatSubpath(path string, removeSingleSlash bool) string {
	if path == "/" {
		if removeSingleSlash {
			return ""
		}
		return path
	}
	return strings.TrimSuffix(path, "/")
}

func (config *Config) MustCorrectURL(section, key, value string) {
	v := config.Section(section).Key(key).String()
	if v == "" {
		v = value
	}
	v = FixFullURL(v)
	config.Section(section).Key(key).SetValue(v)
}

// ExternalDomain returns the Host for the request, using the fixed externalDomain value unless UseProxyHost is true.
func ExternalDomain(gc *gin.Context) string {
	if !UseProxyHost || gc.Request.Host == "" {
		return externalDomain
	}
	return gc.Request.Host
}

// ExternalDomainNoPort attempts to return ExternalDomain() with the port removed. If the internally-used method fails, it is assumed the domain has no port anyway.
func (app *appContext) ExternalDomainNoPort(gc *gin.Context) string {
	domain := ExternalDomain(gc)
	host, _, err := net.SplitHostPort(domain)
	if err != nil {
		return domain
	}
	return host
}

// ExternalURI returns the External URI of jfa-go's root directory (by default, where the admin page is), using the fixed externalURI value unless UseProxyHost is true and gc is not nil.
// When nil is passed, externalURI is returned.
func ExternalURI(gc *gin.Context) string {
	if gc == nil {
		return externalURI
	}

	var proto string
	if gc.Request.TLS != nil || gc.Request.Header.Get("X-Forwarded-Proto") == "https" || gc.Request.Header.Get("X-Forwarded-Protocol") == "https" {
		proto = "https://"
	} else {
		proto = "http://"
	}

	// app.debug.Printf("Request: %+v\n", gc.Request)
	if UseProxyHost && gc.Request.Host != "" {
		return proto + gc.Request.Host + PAGES.Base
	}
	return externalURI
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

	return proto + ExternalDomain(gc) + path
}

// NewConfig reads and patches a config file for use. Passed loggers are used only once. Some dependencies can be reloaded after this is called with ReloadDependents(app).
func NewConfig(configPathOrContents any, dataPath string, logs LoggerSet) (*Config, error) {
	var err error
	config := &Config{}
	config.File, err = ini.ShadowLoad(configPathOrContents)
	if err != nil {
		return config, err
	}

	// URLs
	config.MustSetURLPath("ui", "url_base", "")
	config.MustSetURLPath("url_paths", "admin", "")
	config.MustSetURLPath("url_paths", "user_page", "/my/account")
	config.MustSetURLPath("url_paths", "form", "/invite")
	PAGES.Base = FormatSubpath(config.Section("ui").Key("url_base").String(), true)
	PAGES.Admin = FormatSubpath(config.Section("url_paths").Key("admin").String(), true)
	PAGES.MyAccount = FormatSubpath(config.Section("url_paths").Key("user_page").String(), true)
	PAGES.Form = FormatSubpath(config.Section("url_paths").Key("form").String(), true)
	if !(config.Section("user_page").Key("enabled").MustBool(true)) {
		PAGES.MyAccount = "disabled"
	}
	if PAGES.Base == PAGES.Form || PAGES.Base == "/accounts" || PAGES.Base == "/settings" || PAGES.Base == "/activity" {
		logs.err.Printf(lm.BadURLBase, PAGES.Base)
	}
	logs.info.Printf(lm.SubpathBlockMessage, PAGES.Base, PAGES.Admin, PAGES.MyAccount, PAGES.Form)

	config.MustCorrectURL("jellyfin", "server", "")
	config.MustCorrectURL("jellyfin", "public_server", config.Section("jellyfin").Key("server").String())
	config.MustCorrectURL("ui", "redirect_url", config.Section("jellyfin").Key("public_server").String())

	for _, key := range config.Section("files").Keys() {
		if name := key.Name(); name != "html_templates" && name != "lang_files" {
			key.SetValue(key.MustString(filepath.Join(dataPath, (key.Name() + ".json"))))
		}
	}
	for _, key := range []string{"user_configuration", "user_displayprefs", "user_profiles", "ombi_template", "invites", "emails", "user_template", "custom_emails", "users", "telegram_users", "discord_users", "matrix_users", "announcements", "custom_user_page_content"} {
		config.Section("files").Key(key).SetValue(config.Section("files").Key(key).MustString(filepath.Join(dataPath, (key + ".json"))))
	}
	for _, key := range []string{"matrix_sql"} {
		config.Section("files").Key(key).SetValue(config.Section("files").Key(key).MustString(filepath.Join(dataPath, (key + ".db"))))
	}

	// If true, ExternalDomain() will return one based on the reported Host (ideally reported in "Host" or "X-Forwarded-Host" by the reverse proxy), falling back to externalDomain if not set.
	UseProxyHost = config.Section("ui").Key("use_proxy_host").MustBool(false)
	externalURI = strings.TrimSuffix(strings.TrimSuffix(config.Section("ui").Key("jfa_url").MustString(""), "/invite"), "/")
	if !strings.HasSuffix(externalURI, PAGES.Base) {
		logs.err.Println(lm.NoURLSuffix)
	}
	if externalURI == "" {
		if UseProxyHost {
			logs.err.Println(lm.NoExternalHost + lm.LoginWontSave + lm.SetExternalHostDespiteUseProxyHost)
		} else {
			logs.err.Println(lm.NoExternalHost + lm.LoginWontSave)
		}
	}
	u, err := url.Parse(externalURI)
	if err == nil {
		externalDomain = u.Hostname()
	}

	config.Section("email").Key("no_username").SetValue(strconv.FormatBool(config.Section("email").Key("no_username").MustBool(false)))

	// FIXME: Remove all these, eventually
	// config.MustSetValue("password_resets", "email_html", "jfa-go:"+"password-reset.html")
	// config.MustSetValue("password_resets", "email_text", "jfa-go:"+"password-reset.txt")

	// config.MustSetValue("invite_emails", "email_html", "jfa-go:"+"invite-email.html")
	// config.MustSetValue("invite_emails", "email_text", "jfa-go:"+"invite-email.txt")

	// config.MustSetValue("email_confirmation", "email_html", "jfa-go:"+"confirmation.html")
	// config.MustSetValue("email_confirmation", "email_text", "jfa-go:"+"confirmation.txt")

	// config.MustSetValue("notifications", "expiry_html", "jfa-go:"+"expired.html")
	// config.MustSetValue("notifications", "expiry_text", "jfa-go:"+"expired.txt")

	// config.MustSetValue("notifications", "created_html", "jfa-go:"+"created.html")
	// config.MustSetValue("notifications", "created_text", "jfa-go:"+"created.txt")

	// config.MustSetValue("deletion", "email_html", "jfa-go:"+"deleted.html")
	// config.MustSetValue("deletion", "email_text", "jfa-go:"+"deleted.txt")

	// Deletion template is good enough for these as well.
	// config.MustSetValue("disable_enable", "disabled_html", "jfa-go:"+"deleted.html")
	// config.MustSetValue("disable_enable", "disabled_text", "jfa-go:"+"deleted.txt")
	// config.MustSetValue("disable_enable", "enabled_html", "jfa-go:"+"deleted.html")
	// config.MustSetValue("disable_enable", "enabled_text", "jfa-go:"+"deleted.txt")

	// config.MustSetValue("welcome_email", "email_html", "jfa-go:"+"welcome.html")
	// config.MustSetValue("welcome_email", "email_text", "jfa-go:"+"welcome.txt")

	// config.MustSetValue("template_email", "email_html", "jfa-go:"+"template.html")
	// config.MustSetValue("template_email", "email_text", "jfa-go:"+"template.txt")

	config.MustSetValue("user_expiry", "behaviour", "disable_user")
	// config.MustSetValue("user_expiry", "email_html", "jfa-go:"+"user-expired.html")
	// config.MustSetValue("user_expiry", "email_text", "jfa-go:"+"user-expired.txt")

	// config.MustSetValue("user_expiry", "adjustment_email_html", "jfa-go:"+"expiry-adjusted.html")
	// config.MustSetValue("user_expiry", "adjustment_email_text", "jfa-go:"+"expiry-adjusted.txt")

	// config.MustSetValue("user_expiry", "reminder_email_html", "jfa-go:"+"expiry-reminder.html")
	// config.MustSetValue("user_expiry", "reminder_email_text", "jfa-go:"+"expiry-reminder.txt")

	fnameSettingSuffix := []string{"html", "text"}
	fnameExtension := []string{"html", "txt"}

	for _, cc := range customContent {
		if cc.SourceFile.DefaultValue == "" {
			continue
		}
		for i := range fnameSettingSuffix {
			config.MustSetValue(cc.SourceFile.Section, cc.SourceFile.SettingPrefix+fnameSettingSuffix[i], "jfa-go:"+cc.SourceFile.DefaultValue+"."+fnameExtension[i])
		}
	}

	config.MustSetValue("smtp", "hello_hostname", "localhost")
	config.MustSetValue("smtp", "cert_validation", "true")
	config.MustSetValue("smtp", "auth_type", "4")
	config.MustSetValue("smtp", "port", "465")

	config.MustSetValue("activity_log", "keep_n_records", "1000")
	config.MustSetValue("activity_log", "delete_after_days", "90")

	sc := config.Section("discord").Key("start_command").MustString("start")
	config.Section("discord").Key("start_command").SetValue(strings.TrimPrefix(strings.TrimPrefix(sc, "/"), "!"))

	config.MustSetValue("email", "collect", "true")
	collect := config.Section("email").Key("collect").MustBool(true)
	required := config.Section("email").Key("required").MustBool(false) && collect
	config.Section("email").Key("required").SetValue(strconv.FormatBool(required))
	unique := config.Section("email").Key("require_unique").MustBool(false) && collect
	config.Section("email").Key("require_unique").SetValue(strconv.FormatBool(unique))

	config.MustSetValue("matrix", "topic", "Jellyfin notifications")
	config.MustSetValue("matrix", "show_on_reg", "true")

	config.MustSetValue("discord", "show_on_reg", "true")

	config.MustSetValue("telegram", "show_on_reg", "true")

	config.MustSetValue("backups", "every_n_minutes", "1440")
	config.MustSetValue("backups", "path", filepath.Join(dataPath, "backups"))
	config.MustSetValue("backups", "keep_n_backups", "20")
	config.MustSetValue("backups", "keep_previous_version_backup", "true")

	config.Section("jellyfin").Key("version").SetValue(version)
	config.Section("jellyfin").Key("device").SetValue("jfa-go")
	config.Section("jellyfin").Key("device_id").SetValue(fmt.Sprintf("jfa-go-%s-%s", version, commit))

	config.MustSetValue("jellyfin", "cache_timeout", "30")
	config.MustSetValue("jellyfin", "web_cache_async_timeout", "1")
	config.MustSetValue("jellyfin", "web_cache_sync_timeout", "10")
	config.MustSetValue("jellyfin", "activity_cache_sync_timeout_seconds", "20")

	LOGIP = config.Section("advanced").Key("log_ips").MustBool(false)
	LOGIPU = config.Section("advanced").Key("log_ips_users").MustBool(false)

	config.MustSetValue("advanced", "auth_retry_count", "6")
	config.MustSetValue("advanced", "auth_retry_gap", "10")

	config.MustSetValue("ui", "port", "8056")
	config.MustSetValue("advanced", "tls_port", "8057")

	config.MustSetValue("advanced", "value_log_size", "512")

	pwrMethods := []string{"allow_pwr_username", "allow_pwr_email", "allow_pwr_contact_method"}
	allDisabled := true
	for _, v := range pwrMethods {
		if config.Section("user_page").Key(v).MustBool(true) {
			allDisabled = false
		}
	}
	if allDisabled {
		logs.info.Println(lm.EnableAllPWRMethods)
		for _, v := range pwrMethods {
			config.Section("user_page").Key(v).SetValue("true")
		}
	}

	messagesEnabled = config.Section("messages").Key("enabled").MustBool(false)
	telegramEnabled = config.Section("telegram").Key("enabled").MustBool(false)
	discordEnabled = config.Section("discord").Key("enabled").MustBool(false)
	matrixEnabled = config.Section("matrix").Key("enabled").MustBool(false)
	if !messagesEnabled {
		emailEnabled = false
		telegramEnabled = false
		discordEnabled = false
		matrixEnabled = false
	} else if config.Section("email").Key("method").MustString("") == "" {
		emailEnabled = false
	} else {
		emailEnabled = true
	}
	if !emailEnabled && !telegramEnabled && !discordEnabled && !matrixEnabled {
		messagesEnabled = false
	}

	if proxyEnabled := config.Section("advanced").Key("proxy").MustBool(false); proxyEnabled {
		config.proxyConfig = &easyproxy.ProxyConfig{}
		config.proxyConfig.Protocol = easyproxy.HTTP
		if strings.Contains(config.Section("advanced").Key("proxy_protocol").MustString("http"), "socks") {
			config.proxyConfig.Protocol = easyproxy.SOCKS5
		}
		config.proxyConfig.Addr = config.Section("advanced").Key("proxy_address").MustString("")
		config.proxyConfig.User = config.Section("advanced").Key("proxy_user").MustString("")
		config.proxyConfig.Password = config.Section("advanced").Key("proxy_password").MustString("")
		config.proxyTransport, err = easyproxy.NewTransport(*(config.proxyConfig))
		if err != nil {
			logs.err.Printf(lm.FailedInitProxy, config.proxyConfig.Addr, err)
			// As explained in lm.FailedInitProxy, sleep here might grab the admin's attention,
			// Since we don't crash on this failing.
			time.Sleep(15 * time.Second)
			config.proxyConfig = nil
			config.proxyTransport = nil
		} else {
			logs.info.Printf(lm.InitProxy, config.proxyConfig.Addr)
		}
	}

	config.MustSetValue("updates", "enabled", "true")

	substituteStrings = config.Section("jellyfin").Key("substitute_jellyfin_strings").MustString("")

	if substituteStrings != "" {
		v := config.Section("ui").Key("success_message")
		v.SetValue(strings.ReplaceAll(v.String(), "Jellyfin", substituteStrings))
	}

	datePattern = config.Section("messages").Key("date_format").String()
	timePattern = `%H:%M`
	if !(config.Section("messages").Key("use_24h").MustBool(true)) {
		timePattern = `%I:%M %p`
	}

	return config, nil
}

// ReloadDependents re-initialises or applies changes to components of the app which can be reconfigured without restarting.
func (config *Config) ReloadDependents(app *appContext) {
	oldFormLang := config.Section("ui").Key("language").MustString("")
	if oldFormLang != "" {
		app.storage.lang.chosenUserLang = oldFormLang
	}
	newFormLang := config.Section("ui").Key("language-form").MustString("")
	if newFormLang != "" {
		app.storage.lang.chosenUserLang = newFormLang
	}

	app.storage.lang.chosenAdminLang = config.Section("ui").Key("language-admin").MustString("en-us")
	app.storage.lang.chosenEmailLang = config.Section("email").Key("language").MustString("en-us")
	app.storage.lang.chosenPWRLang = config.Section("password_resets").Key("language").MustString("en-us")
	app.storage.lang.chosenTelegramLang = config.Section("telegram").Key("language").MustString("en-us")

	releaseChannel := config.Section("updates").Key("channel").String()
	if config.Section("updates").Key("enabled").MustBool(false) {
		v := version
		if releaseChannel == "stable" {
			if version == "git" {
				v = "0.0.0"
			}
		} else if releaseChannel == "unstable" {
			v = "git"
		}
		app.updater = NewUpdater(baseURL, namespace, repo, v, commit, updater)
		if config.proxyTransport != nil {
			app.updater.SetTransport(config.proxyTransport)
		}
	}
	if releaseChannel == "" {
		if version == "git" {
			releaseChannel = "unstable"
		} else {
			releaseChannel = "stable"
		}
		config.MustSetValue("updates", "channel", releaseChannel)
	}

	app.email = NewEmailer(config, app.storage, app.LoggerSet)

}

func (app *appContext) ReloadConfig() {
	var err error = nil
	app.config, err = NewConfig(app.configPath, app.dataPath, app.LoggerSet)
	if err != nil {
		app.err.Fatalf(lm.FailedLoadConfig, app.configPath, err)
	}

	app.config.ReloadDependents(app)
	app.info.Printf(lm.LoadConfig, app.configPath)
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
