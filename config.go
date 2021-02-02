package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

var emailEnabled = false

func (app *appContext) GetPath(sect, key string) (fs.FS, string) {
	val := app.config.Section(sect).Key(key).MustString("")
	if strings.HasPrefix(val, "jfa-go:") {
		return localFS, strings.TrimPrefix(val, "jfa-go:")
	}
	return app.systemFS, val
}

func (app *appContext) loadConfig() error {
	var err error
	app.config, err = ini.Load(app.configPath)
	if err != nil {
		return err
	}

	app.config.Section("jellyfin").Key("public_server").SetValue(app.config.Section("jellyfin").Key("public_server").MustString(app.config.Section("jellyfin").Key("server").String()))

	for _, key := range app.config.Section("files").Keys() {
		if key.Name() != "html_templates" {
			key.SetValue(key.MustString(filepath.Join(app.dataPath, (key.Name() + ".json"))))
		}
	}
	for _, key := range []string{"user_configuration", "user_displayprefs", "user_profiles", "ombi_template", "invites", "emails", "user_template"} {
		app.config.Section("files").Key(key).SetValue(app.config.Section("files").Key(key).MustString(filepath.Join(app.dataPath, (key + ".json"))))
	}
	app.URLBase = strings.TrimSuffix(app.config.Section("ui").Key("url_base").MustString(""), "/")
	app.config.Section("email").Key("no_username").SetValue(strconv.FormatBool(app.config.Section("email").Key("no_username").MustBool(false)))

	app.config.Section("password_resets").Key("email_html").SetValue(app.config.Section("password_resets").Key("email_html").MustString("jfa-go:" + "email.html"))
	app.config.Section("password_resets").Key("email_text").SetValue(app.config.Section("password_resets").Key("email_text").MustString("jfa-go:" + "email.txt"))

	app.config.Section("invite_emails").Key("email_html").SetValue(app.config.Section("invite_emails").Key("email_html").MustString("jfa-go:" + "invite-email.html"))
	app.config.Section("invite_emails").Key("email_text").SetValue(app.config.Section("invite_emails").Key("email_text").MustString("jfa-go:" + "invite-email.txt"))

	app.config.Section("email_confirmation").Key("email_html").SetValue(app.config.Section("email_confirmation").Key("email_html").MustString("jfa-go:" + "confirmation.html"))
	app.config.Section("email_confirmation").Key("email_text").SetValue(app.config.Section("email_confirmation").Key("email_text").MustString("jfa-go:" + "confirmation.txt"))

	app.config.Section("notifications").Key("expiry_html").SetValue(app.config.Section("notifications").Key("expiry_html").MustString("jfa-go:" + "expired.html"))
	app.config.Section("notifications").Key("expiry_text").SetValue(app.config.Section("notifications").Key("expiry_text").MustString("jfa-go:" + "expired.txt"))

	app.config.Section("notifications").Key("created_html").SetValue(app.config.Section("notifications").Key("created_html").MustString("jfa-go:" + "created.html"))
	app.config.Section("notifications").Key("created_text").SetValue(app.config.Section("notifications").Key("created_text").MustString("jfa-go:" + "created.txt"))

	app.config.Section("deletion").Key("email_html").SetValue(app.config.Section("deletion").Key("email_html").MustString("jfa-go:" + "deleted.html"))
	app.config.Section("deletion").Key("email_text").SetValue(app.config.Section("deletion").Key("email_text").MustString("jfa-go:" + "deleted.txt"))

	app.config.Section("welcome_email").Key("email_html").SetValue(app.config.Section("welcome_email").Key("email_html").MustString("jfa-go:" + "welcome.html"))
	app.config.Section("welcome_email").Key("email_text").SetValue(app.config.Section("welcome_email").Key("email_text").MustString("jfa-go:" + "welcome.txt"))

	app.config.Section("jellyfin").Key("version").SetValue(VERSION)
	app.config.Section("jellyfin").Key("device").SetValue("jfa-go")
	app.config.Section("jellyfin").Key("device_id").SetValue(fmt.Sprintf("jfa-go-%s-%s", VERSION, COMMIT))

	if app.config.Section("email").Key("method").MustString("") == "" {
		emailEnabled = false
	} else {
		emailEnabled = true
	}

	substituteStrings = app.config.Section("jellyfin").Key("substitute_jellyfin_strings").MustString("")

	oldFormLang := app.config.Section("ui").Key("language").MustString("")
	if oldFormLang != "" {
		app.storage.lang.chosenFormLang = oldFormLang
	}
	newFormLang := app.config.Section("ui").Key("language-form").MustString("")
	if newFormLang != "" {
		app.storage.lang.chosenFormLang = newFormLang
	}
	app.storage.lang.chosenAdminLang = app.config.Section("ui").Key("language-admin").MustString("en-us")
	app.storage.lang.chosenEmailLang = app.config.Section("email").Key("language").MustString("en-us")

	app.email = NewEmailer(app)

	return nil
}
