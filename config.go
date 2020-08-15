package main

import (
	"gopkg.in/ini.v1"
	"path/filepath"
	"strconv"
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

func (ctx *appContext) loadDefaults() (err error) {
	var cfb []byte
	cfb, err = ioutil.ReadFile(ctx.configBase_path)
	if err != nil {
		return
	}
	json.Unmarshal(cfb, ctx.defaults)
	return
}*/

func (ctx *appContext) loadConfig() error {
	var err error
	ctx.config, err = ini.Load(ctx.config_path)
	if err != nil {
		return err
	}

	ctx.config.Section("jellyfin").Key("public_server").SetValue(ctx.config.Section("jellyfin").Key("public_server").MustString(ctx.config.Section("jellyfin").Key("server").String()))

	for _, key := range ctx.config.Section("files").Keys() {
		// if key.MustString("") == "" && key.Name() != "custom_css" {
		// 	key.SetValue(filepath.Join(ctx.data_path, (key.Name() + ".json")))
		// }
		key.SetValue(key.MustString(filepath.Join(ctx.data_path, (key.Name() + ".json"))))
	}
	for _, key := range []string{"user_configuration", "user_displayprefs"} {
		// if ctx.config.Section("files").Key(key).MustString("") == "" {
		// 	key.SetValue(filepath.Join(ctx.data_path, (key.Name() + ".json")))
		// }
		ctx.config.Section("files").Key(key).SetValue(ctx.config.Section("files").Key(key).MustString(filepath.Join(ctx.data_path, (key + ".json"))))
	}

	ctx.config.Section("email").Key("no_username").SetValue(strconv.FormatBool(ctx.config.Section("email").Key("no_username").MustBool(false)))

	ctx.config.Section("password_resets").Key("email_html").SetValue(ctx.config.Section("password_resets").Key("email_html").MustString(filepath.Join(ctx.local_path, "email.html")))
	ctx.config.Section("password_resets").Key("email_text").SetValue(ctx.config.Section("password_resets").Key("email_text").MustString(filepath.Join(ctx.local_path, "email.txt")))

	ctx.config.Section("invite_emails").Key("email_html").SetValue(ctx.config.Section("invite_emails").Key("email_html").MustString(filepath.Join(ctx.local_path, "invite-email.html")))
	ctx.config.Section("invite_emails").Key("email_text").SetValue(ctx.config.Section("invite_emails").Key("email_text").MustString(filepath.Join(ctx.local_path, "invite-email.txt")))

	ctx.config.Section("notifications").Key("expiry_html").SetValue(ctx.config.Section("notifications").Key("expiry_html").MustString(filepath.Join(ctx.local_path, "expired.html")))
	ctx.config.Section("notifications").Key("expiry_text").SetValue(ctx.config.Section("notifications").Key("expiry_text").MustString(filepath.Join(ctx.local_path, "expired.txt")))

	ctx.config.Section("notifications").Key("created_html").SetValue(ctx.config.Section("notifications").Key("created_html").MustString(filepath.Join(ctx.local_path, "created.html")))
	ctx.config.Section("notifications").Key("created_text").SetValue(ctx.config.Section("notifications").Key("created_text").MustString(filepath.Join(ctx.local_path, "created.txt")))

	return nil
}
