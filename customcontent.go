package main

import (
	"fmt"
	"maps"
	"slices"
)

func defaultVars(vars ...string) []string {
	return slices.Concat(vars, []string{
		"username",
	})
}

func defaultVals(vals map[string]any) map[string]any {
	maps.Copy(vals, map[string]any{
		"username": "Username",
	})
	return vals
}

var customContent = map[string]CustomContentInfo{
	"EmailConfirmation": {
		Name:        "EmailConfirmation",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].EmailConfirmation["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("email_confirmation").Key("subject").MustString(lang.EmailConfirmation.get("title"))
		},
		Variables: defaultVars(
			"confirmationURL",
		),
		Placeholders: defaultVals(map[string]any{
			"confirmationURL": "https://sub2.test.url/invite/xxxxxx?key=xxxxxx",
		}),
		SourceFile: ContentSourceFileInfo{
			Section:       "email_confirmation",
			SettingPrefix: "email_",
			DefaultValue:  "confirmation",
		},
	},
	"ExpiryReminder": {
		Name:        "ExpiryReminder",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].ExpiryReminder["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("user_expiry").Key("reminder_subject").MustString(lang.ExpiryReminder.get("title"))
		},
		Variables: defaultVars(
			"expiresIn",
			"date",
			"time",
		),
		Placeholders: defaultVals(map[string]any{
			"expiresIn": "3d",
			"date":      "20/08/25",
			"time":      "14:19",
		}),
		SourceFile: ContentSourceFileInfo{
			Section:       "user_expiry",
			SettingPrefix: "reminder_email_",
			DefaultValue:  "expiry-reminder",
		},
	},
	"InviteEmail": {
		Name:        "InviteEmail",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].InviteEmail["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("invite_emails").Key("subject").MustString(lang.InviteEmail.get("title"))
		},
		Variables: []string{
			"date",
			"time",
			"expiresInMinutes",
			"inviteURL",
		},
		Placeholders: defaultVals(map[string]any{
			"date":             "01/01/01",
			"time":             "00:00",
			"expiresInMinutes": "16d 13h 19m",
			"inviteURL":        "https://sub2.test.url/invite/xxxxxx",
		}),
		SourceFile: ContentSourceFileInfo{
			Section:       "invite_emails",
			SettingPrefix: "email_",
			DefaultValue:  "invite-email",
		},
	},
	"InviteExpiry": {
		Name:        "InviteExpiry",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].InviteExpiry["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return lang.InviteExpiry.get("title")
		},
		Variables: []string{
			"code",
			"time",
		},
		Placeholders: map[string]any{
			"code": "\"xxxxxx\"",
			"time": "01/01/01 00:00",
		},
		SourceFile: ContentSourceFileInfo{
			Section:       "notifications",
			SettingPrefix: "expiry_",
			DefaultValue:  "expired",
		},
	},
	"PasswordReset": {
		Name:        "PasswordReset",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].PasswordReset["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("password_resets").Key("subject").MustString(lang.PasswordReset.get("title"))
		},
		Variables: defaultVars(
			"date",
			"time",
			"expiresInMinutes",
			"pin",
		),
		Placeholders: defaultVals(map[string]any{
			"date":             "01/01/01",
			"time":             "00:00",
			"expiresInMinutes": "16d 13h 19m",
			"pin":              "12-34-56",
		}),
		SourceFile: ContentSourceFileInfo{
			Section:       "password_resets",
			SettingPrefix: "email_",
			// This was the first email type added, hence the undescriptive filename.
			DefaultValue: "email",
		},
	},
	"UserCreated": {
		Name:        "UserCreated",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].UserCreated["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return lang.UserCreated.get("title")
		},
		Variables: []string{
			"code",
			"name",
			"address",
			"time",
		},
		Placeholders: map[string]any{
			"name":    "Subject Username",
			"code":    "\"xxxxxx\"",
			"address": "Email Address",
			"time":    "01/01/01 00:00",
		},
		SourceFile: ContentSourceFileInfo{
			Section:       "notifications",
			SettingPrefix: "created_",
			DefaultValue:  "created",
		},
	},
	"UserDeleted": {
		Name:        "UserDeleted",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].UserDeleted["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("deletion").Key("subject").MustString(lang.UserDeleted.get("title"))
		},
		Variables: defaultVars(
			"reason",
		),
		Placeholders: defaultVals(map[string]any{
			"reason": "Reason",
		}),
		SourceFile: ContentSourceFileInfo{
			Section:       "deletion",
			SettingPrefix: "email_",
			DefaultValue:  "deleted",
		},
	},
	"UserDisabled": {
		Name:        "UserDisabled",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].UserDisabled["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("disable_enable").Key("subject_disabled").MustString(lang.UserDisabled.get("title"))
		},
		Variables: defaultVars(
			"reason",
		),
		Placeholders: defaultVals(map[string]any{
			"reason": "Reason",
		}),
		SourceFile: ContentSourceFileInfo{
			Section:       "disable_enable",
			SettingPrefix: "disabled_",
			// Template is shared between deletion enabling and disabling.
			DefaultValue: "deleted",
		},
	},
	"UserEnabled": {
		Name:        "UserEnabled",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].UserEnabled["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("disable_enable").Key("subject_enabled").MustString(lang.UserEnabled.get("title"))
		},
		Variables: defaultVars(
			"reason",
		),
		Placeholders: defaultVals(map[string]any{
			"reason": "Reason",
		}),
		SourceFile: ContentSourceFileInfo{
			Section:       "disable_enable",
			SettingPrefix: "enabled_",
			// Template is shared between deletion enabling and disabling.
			DefaultValue: "deleted",
		},
	},
	"UserExpired": {
		Name:        "UserExpired",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].UserExpired["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("user_expiry").Key("subject").MustString(lang.UserExpired.get("title"))
		},
		Variables:    defaultVars(),
		Placeholders: defaultVals(map[string]any{}),
		SourceFile: ContentSourceFileInfo{
			Section:       "user_expiry",
			SettingPrefix: "email_",
			DefaultValue:  "user-expired",
		},
	},
	"UserExpiryAdjusted": {
		Name:        "UserExpiryAdjusted",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].UserExpiryAdjusted["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("user_expiry").Key("adjustment_subject").MustString(lang.UserExpiryAdjusted.get("title"))
		},
		Variables: defaultVars(
			"newExpiry",
			"reason",
		),
		Placeholders: defaultVals(map[string]any{
			"newExpiry": "",
			"reason":    "Reason",
		}),
		SourceFile: ContentSourceFileInfo{
			Section:       "user_expiry",
			SettingPrefix: "adjustment_email_",
			DefaultValue:  "expiry-adjusted",
		},
	},
	"WelcomeEmail": {
		Name:        "WelcomeEmail",
		ContentType: CustomMessage,
		DisplayName: func(dict *Lang, lang string) string { return dict.Email[lang].WelcomeEmail["name"] },
		Subject: func(config *Config, lang *emailLang) string {
			return config.Section("welcome_email").Key("subject").MustString(lang.WelcomeEmail.get("title"))
		},
		Variables: defaultVars(
			"jellyfinURL",
			"yourAccountWillExpire",
		),
		Conditionals: []string{
			"yourAccountWillExpire",
		},
		Placeholders: defaultVals(map[string]any{
			"jellyfinURL":           "https://example.io",
			"yourAccountWillExpire": "17/08/25 14:19",
		}),
		SourceFile: ContentSourceFileInfo{
			Section:       "welcome_email",
			SettingPrefix: "email_",
			DefaultValue:  "welcome",
		},
	},
	"TemplateEmail": {
		Name: "TemplateEmail",
		DisplayName: func(dict *Lang, lang string) string {
			return "EmptyCustomContent"
		},
		ContentType: CustomTemplate,
		SourceFile: ContentSourceFileInfo{
			Section:       "template_email",
			SettingPrefix: "email_",
			DefaultValue:  "template",
		},
	},
	"UserLogin": {
		Name:        "UserLogin",
		ContentType: CustomCard,
		DisplayName: func(dict *Lang, lang string) string {
			if _, ok := dict.Admin[lang]; !ok {
				lang = dict.chosenAdminLang
			}
			return dict.Admin[lang].Strings["userPageLogin"]
		},
		Variables: []string{},
	},
	"UserPage": {
		Name:        "UserPage",
		ContentType: CustomCard,
		DisplayName: func(dict *Lang, lang string) string {
			if _, ok := dict.Admin[lang]; !ok {
				lang = dict.chosenAdminLang
			}
			return dict.Admin[lang].Strings["userPagePage"]
		},
		Variables:    defaultVars(),
		Placeholders: defaultVals(map[string]any{}),
	},
	"PostSignupCard": {
		Name:        "PostSignupCard",
		ContentType: CustomCard,
		DisplayName: func(dict *Lang, lang string) string {
			if _, ok := dict.Admin[lang]; !ok {
				lang = dict.chosenAdminLang
			}
			return dict.Admin[lang].Strings["postSignupCard"]
		},
		Description: func(dict *Lang, lang string) string {
			if _, ok := dict.Admin[lang]; !ok {
				lang = dict.chosenAdminLang
			}
			return dict.Admin[lang].Strings["postSignupCardDescription"]
		},
		Variables: defaultVars(
			"myAccountURL",
		),
		Placeholders: defaultVals(map[string]any{
			"myAccountURL": "https://sub2.test.url/my/account",
		}),
	},
}

var EmptyCustomContent = CustomContentInfo{
	Name:        "EmptyCustomContent",
	ContentType: CustomMessage,
	DisplayName: func(dict *Lang, lang string) string {
		return "EmptyCustomContent"
	},
	Subject: func(config *Config, lang *emailLang) string {
		return "EmptyCustomContent"
	},
	Description:  nil,
	Variables:    []string{},
	Placeholders: map[string]any{},
}

var AnnouncementCustomContent = func(subject string) CustomContentInfo {
	cci := EmptyCustomContent
	cci.Subject = func(config *Config, lang *emailLang) string { return subject }
	cci.Variables = defaultVars()
	cci.Placeholders = defaultVals(map[string]any{})
	return cci
}

var _runtimeValidation = func() bool {
	for name, cc := range customContent {
		if name != cc.Name {
			panic(fmt.Errorf("customContent key and name not matching: %s != %s", name, cc.Name))
		}
		if cc.DisplayName == nil {
			panic(fmt.Errorf("no customContent[%s] DisplayName set", name))
		}
	}
	return true
}()
