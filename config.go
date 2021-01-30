package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

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

	app.config.Section("password_resets").Key("email_html").SetValue(app.config.Section("password_resets").Key("email_html").MustString(filepath.Join(app.localPath, "email.html")))
	app.config.Section("password_resets").Key("email_text").SetValue(app.config.Section("password_resets").Key("email_text").MustString(filepath.Join(app.localPath, "email.txt")))

	app.config.Section("invite_emails").Key("email_html").SetValue(app.config.Section("invite_emails").Key("email_html").MustString(filepath.Join(app.localPath, "invite-email.html")))
	app.config.Section("invite_emails").Key("email_text").SetValue(app.config.Section("invite_emails").Key("email_text").MustString(filepath.Join(app.localPath, "invite-email.txt")))

	app.config.Section("email_confirmation").Key("email_html").SetValue(app.config.Section("email_confirmation").Key("email_html").MustString(filepath.Join(app.localPath, "confirmation.html")))
	fmt.Println(app.config.Section("email_confirmation").Key("email_html").String())
	app.config.Section("email_confirmation").Key("email_text").SetValue(app.config.Section("email_confirmation").Key("email_text").MustString(filepath.Join(app.localPath, "confirmation.txt")))

	app.config.Section("notifications").Key("expiry_html").SetValue(app.config.Section("notifications").Key("expiry_html").MustString(filepath.Join(app.localPath, "expired.html")))
	app.config.Section("notifications").Key("expiry_text").SetValue(app.config.Section("notifications").Key("expiry_text").MustString(filepath.Join(app.localPath, "expired.txt")))

	app.config.Section("notifications").Key("created_html").SetValue(app.config.Section("notifications").Key("created_html").MustString(filepath.Join(app.localPath, "created.html")))
	app.config.Section("notifications").Key("created_text").SetValue(app.config.Section("notifications").Key("created_text").MustString(filepath.Join(app.localPath, "created.txt")))

	app.config.Section("deletion").Key("email_html").SetValue(app.config.Section("deletion").Key("email_html").MustString(filepath.Join(app.localPath, "deleted.html")))
	app.config.Section("deletion").Key("email_text").SetValue(app.config.Section("deletion").Key("email_text").MustString(filepath.Join(app.localPath, "deleted.txt")))

	app.config.Section("welcome_email").Key("email_html").SetValue(app.config.Section("welcome_email").Key("email_html").MustString(filepath.Join(app.localPath, "welcome.html")))
	app.config.Section("welcome_email").Key("email_text").SetValue(app.config.Section("welcome_email").Key("email_text").MustString(filepath.Join(app.localPath, "welcome.txt")))

	app.config.Section("jellyfin").Key("version").SetValue(VERSION)
	app.config.Section("jellyfin").Key("device").SetValue("jfa-go")
	app.config.Section("jellyfin").Key("device_id").SetValue(fmt.Sprintf("jfa-go-%s-%s", VERSION, COMMIT))

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
