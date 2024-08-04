package main

import (
	"time"

	"github.com/dgraph-io/badger/v3"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/hrfee/mediabrowser"
	"github.com/timshannon/badgerhold/v4"
)

// clearEmails removes stored emails for users which no longer exist.
// meant to be called with other such housekeeping functions, so assumes
// the user cache is fresh.
func (app *appContext) clearEmails() {
	app.debug.Println(lm.HousekeepingEmail)
	emails := app.storage.GetEmails()
	for _, email := range emails {
		_, _, err := app.jf.UserByID(email.JellyfinID, false)
		// Make sure the user doesn't exist, and no other error has occured
		switch err.(type) {
		case mediabrowser.ErrUserNotFound:
			app.storage.DeleteEmailsKey(email.JellyfinID)
		default:
			continue
		}
	}
}

// clearDiscord does the same as clearEmails, but for Discord Users.
func (app *appContext) clearDiscord() {
	app.debug.Println(lm.HousekeepingDiscord)
	discordUsers := app.storage.GetDiscord()
	for _, discordUser := range discordUsers {
		_, _, err := app.jf.UserByID(discordUser.JellyfinID, false)
		// Make sure the user doesn't exist, and no other error has occured
		switch err.(type) {
		case mediabrowser.ErrUserNotFound:
			// Remove role in case their account was deleted oustide of jfa-go
			app.discord.RemoveRole(discordUser.MethodID().(string))
			app.storage.DeleteDiscordKey(discordUser.JellyfinID)
		default:
			continue
		}
	}
}

// clearMatrix does the same as clearEmails, but for Matrix Users.
func (app *appContext) clearMatrix() {
	app.debug.Println(lm.HousekeepingMatrix)
	matrixUsers := app.storage.GetMatrix()
	for _, matrixUser := range matrixUsers {
		_, _, err := app.jf.UserByID(matrixUser.JellyfinID, false)
		// Make sure the user doesn't exist, and no other error has occured
		switch err.(type) {
		case mediabrowser.ErrUserNotFound:
			app.storage.DeleteMatrixKey(matrixUser.JellyfinID)
		default:
			continue
		}
	}
}

// clearTelegram does the same as clearEmails, but for Telegram Users.
func (app *appContext) clearTelegram() {
	app.debug.Println(lm.HousekeepingTelegram)
	telegramUsers := app.storage.GetTelegram()
	for _, telegramUser := range telegramUsers {
		_, _, err := app.jf.UserByID(telegramUser.JellyfinID, false)
		// Make sure the user doesn't exist, and no other error has occured
		switch err.(type) {
		case mediabrowser.ErrUserNotFound:
			app.storage.DeleteTelegramKey(telegramUser.JellyfinID)
		default:
			continue
		}
	}
}

func (app *appContext) clearPWRCaptchas() {
	app.debug.Println(lm.HousekeepingCaptcha)
	captchas := map[string]Captcha{}
	for k, capt := range app.pwrCaptchas {
		if capt.Generated.Add(CAPTCHA_VALIDITY * time.Second).After(time.Now()) {
			captchas[k] = capt
		}
	}
	app.pwrCaptchas = captchas
}

func (app *appContext) clearActivities() {
	app.debug.Println(lm.HousekeepingActivity)
	keepCount := app.config.Section("activity_log").Key("keep_n_records").MustInt(1000)
	maxAgeDays := app.config.Section("activity_log").Key("delete_after_days").MustInt(90)
	minAge := time.Now().AddDate(0, 0, -maxAgeDays)
	err := error(nil)
	errorSource := 0
	if maxAgeDays != 0 {
		err = app.storage.db.DeleteMatching(&Activity{}, badgerhold.Where("Time").Lt(minAge))
	}
	if err == nil && keepCount != 0 {
		// app.debug.Printf("Keeping %d records", keepCount)
		err = app.storage.db.DeleteMatching(&Activity{}, (&badgerhold.Query{}).Reverse().SortBy("Time").Skip(keepCount))
		if err != nil {
			errorSource = 1
		}
	}
	if err == badger.ErrTxnTooBig {
		app.debug.Printf(lm.ActivityLogTxnTooBig)
		list := []Activity{}
		if errorSource == 0 {
			app.storage.db.Find(&list, badgerhold.Where("Time").Lt(minAge))
		} else {
			app.storage.db.Find(&list, (&badgerhold.Query{}).Reverse().SortBy("Time").Skip(keepCount))
		}
		for _, record := range list {
			app.storage.DeleteActivityKey(record.ID)
		}
	}
}

func newHousekeepingDaemon(interval time.Duration, app *appContext) *GenericDaemon {
	d := NewGenericDaemon(interval, app,
		func(app *appContext) {
			app.debug.Println(lm.HousekeepingInvites)
			app.checkInvites()
		},
		func(app *appContext) { app.clearActivities() },
	)

	d.Name("Housekeeping daemon")

	clearEmail := app.config.Section("email").Key("require_unique").MustBool(false)
	clearDiscord := app.config.Section("discord").Key("require_unique").MustBool(false) || app.config.Section("discord").Key("disable_enable_role").MustBool(false)
	clearTelegram := app.config.Section("telegram").Key("require_unique").MustBool(false)
	clearMatrix := app.config.Section("matrix").Key("require_unique").MustBool(false)
	clearPWR := app.config.Section("captcha").Key("enabled").MustBool(false) && !app.config.Section("captcha").Key("recaptcha").MustBool(false)

	if clearEmail || clearDiscord || clearTelegram || clearMatrix {
		d.appendJobs(func(app *appContext) { app.jf.CacheExpiry = time.Now() })
	}

	if clearEmail {
		d.appendJobs(func(app *appContext) { app.clearEmails() })
	}
	if clearDiscord {
		d.appendJobs(func(app *appContext) { app.clearDiscord() })
	}
	if clearTelegram {
		d.appendJobs(func(app *appContext) { app.clearTelegram() })
	}
	if clearMatrix {
		d.appendJobs(func(app *appContext) { app.clearMatrix() })
	}
	if clearPWR {
		d.appendJobs(func(app *appContext) { app.clearPWRCaptchas() })
	}

	return d
}
