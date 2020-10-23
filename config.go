package main

import (
	"fmt"
	"path/filepath"
	"strconv"

	"gopkg.in/ini.v1"
)

/*var DeCamel ini.NameMapper = func(raw string) string {
	out := make([]rune, 0, len(raw))
	upper := 0
	for _, c := range raw {
		if unicode.IsUpper(c) {
			upper++
		}
		if upper == 2 {
			out = append(out, '_')
			upper = 0
		}
		out = append(out, unicode.ToLower(c))
	}
	return string(out)
}

func (app *appContext) loadDefaults() (err error) {
	var cfb []byte
	cfb, err = ioutil.ReadFile(app.configBase_path)
	if err != nil {
		return
	}
	json.Unmarshal(cfb, app.defaults)
	return
}*/

func (app *appContext) loadConfig() error {
	var err error
	app.config, err = ini.Load(app.config_path)
	if err != nil {
		return err
	}

	app.config.Section("jellyfin").Key("public_server").SetValue(app.config.Section("jellyfin").Key("public_server").MustString(app.config.Section("jellyfin").Key("server").String()))

	for _, key := range app.config.Section("files").Keys() {
		// if key.MustString("") == "" && key.Name() != "custom_css" {
		// 	key.SetValue(filepath.Join(app.data_path, (key.Name() + ".json")))
		// }
		if key.Name() != "html_templates" {
			key.SetValue(key.MustString(filepath.Join(app.data_path, (key.Name() + ".json"))))
		}
	}
	for _, key := range []string{"user_configuration", "user_displayprefs", "user_profiles", "ombi_template"} {
		// if app.config.Section("files").Key(key).MustString("") == "" {
		// 	key.SetValue(filepath.Join(app.data_path, (key.Name() + ".json")))
		// }
		app.config.Section("files").Key(key).SetValue(app.config.Section("files").Key(key).MustString(filepath.Join(app.data_path, (key + ".json"))))
	}

	app.config.Section("email").Key("no_username").SetValue(strconv.FormatBool(app.config.Section("email").Key("no_username").MustBool(false)))

	app.config.Section("password_resets").Key("email_html").SetValue(app.config.Section("password_resets").Key("email_html").MustString(filepath.Join(app.local_path, "email.html")))
	app.config.Section("password_resets").Key("email_text").SetValue(app.config.Section("password_resets").Key("email_text").MustString(filepath.Join(app.local_path, "email.txt")))

	app.config.Section("invite_emails").Key("email_html").SetValue(app.config.Section("invite_emails").Key("email_html").MustString(filepath.Join(app.local_path, "invite-email.html")))
	app.config.Section("invite_emails").Key("email_text").SetValue(app.config.Section("invite_emails").Key("email_text").MustString(filepath.Join(app.local_path, "invite-email.txt")))

	app.config.Section("notifications").Key("expiry_html").SetValue(app.config.Section("notifications").Key("expiry_html").MustString(filepath.Join(app.local_path, "expired.html")))
	app.config.Section("notifications").Key("expiry_text").SetValue(app.config.Section("notifications").Key("expiry_text").MustString(filepath.Join(app.local_path, "expired.txt")))

	app.config.Section("notifications").Key("created_html").SetValue(app.config.Section("notifications").Key("created_html").MustString(filepath.Join(app.local_path, "created.html")))
	app.config.Section("notifications").Key("created_text").SetValue(app.config.Section("notifications").Key("created_text").MustString(filepath.Join(app.local_path, "created.txt")))

	app.config.Section("deletion").Key("email_html").SetValue(app.config.Section("deletion").Key("email_html").MustString(filepath.Join(app.local_path, "deleted.html")))
	app.config.Section("deletion").Key("email_text").SetValue(app.config.Section("deletion").Key("email_text").MustString(filepath.Join(app.local_path, "deleted.txt")))

	app.config.Section("jellyfin").Key("version").SetValue(VERSION)
	app.config.Section("jellyfin").Key("device").SetValue("jfa-go")
	app.config.Section("jellyfin").Key("device_id").SetValue(fmt.Sprintf("jfa-go-%s-%s", VERSION, COMMIT))

	app.email = NewEmailer(app)

	return nil
}
