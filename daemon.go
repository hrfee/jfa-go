package main

import (
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/hrfee/mediabrowser"
	"github.com/timshannon/badgerhold/v4"
)

// clearEmails removes stored emails for users which no longer exist.
// meant to be called with other such housekeeping functions, so assumes
// the user cache is fresh.
func (app *appContext) clearEmails() {
	app.debug.Println("Housekeeping: removing unused email addresses")
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
	app.debug.Println("Housekeeping: removing unused Discord IDs")
	discordUsers := app.storage.GetDiscord()
	for _, discordUser := range discordUsers {
		_, _, err := app.jf.UserByID(discordUser.JellyfinID, false)
		// Make sure the user doesn't exist, and no other error has occured
		switch err.(type) {
		case mediabrowser.ErrUserNotFound:
			app.storage.DeleteDiscordKey(discordUser.JellyfinID)
		default:
			continue
		}
	}
}

// clearMatrix does the same as clearEmails, but for Matrix Users.
func (app *appContext) clearMatrix() {
	app.debug.Println("Housekeeping: removing unused Matrix IDs")
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
	app.debug.Println("Housekeeping: removing unused Telegram IDs")
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
	app.debug.Println("Housekeeping: Clearing old PWR Captchas")
	captchas := map[string]Captcha{}
	for k, capt := range app.pwrCaptchas {
		if capt.Generated.Add(CAPTCHA_VALIDITY * time.Second).After(time.Now()) {
			captchas[k] = capt
		}
	}
	app.pwrCaptchas = captchas
}

func (app *appContext) clearActivities() {
	app.debug.Println("Housekeeping: Cleaning up Activity log...")
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
		app.debug.Printf("Activities: Delete txn was too big, doing it manually.")
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

// https://bbengfort.github.io/snippets/2016/06/26/background-work-goroutines-timer.html THANKS

type housekeepingDaemon struct {
	Stopped         bool
	ShutdownChannel chan string
	Interval        time.Duration
	period          time.Duration
	jobs            []func(app *appContext)
	app             *appContext
}

func newInviteDaemon(interval time.Duration, app *appContext) *housekeepingDaemon {
	daemon := housekeepingDaemon{
		Stopped:         false,
		ShutdownChannel: make(chan string),
		Interval:        interval,
		period:          interval,
		app:             app,
	}
	daemon.jobs = []func(app *appContext){
		func(app *appContext) {
			app.debug.Println("Housekeeping: Checking for expired invites")
			app.checkInvites()
		},
		func(app *appContext) { app.clearActivities() },
	}

	clearEmail := app.config.Section("email").Key("require_unique").MustBool(false)
	clearDiscord := app.config.Section("discord").Key("require_unique").MustBool(false)
	clearTelegram := app.config.Section("telegram").Key("require_unique").MustBool(false)
	clearMatrix := app.config.Section("matrix").Key("require_unique").MustBool(false)
	clearPWR := app.config.Section("captcha").Key("enabled").MustBool(false) && !app.config.Section("captcha").Key("recaptcha").MustBool(false)

	if clearEmail || clearDiscord || clearTelegram || clearMatrix {
		daemon.jobs = append(daemon.jobs, func(app *appContext) { app.jf.CacheExpiry = time.Now() })
	}

	if clearEmail {
		daemon.jobs = append(daemon.jobs, func(app *appContext) { app.clearEmails() })
	}
	if clearDiscord {
		daemon.jobs = append(daemon.jobs, func(app *appContext) { app.clearDiscord() })
	}
	if clearTelegram {
		daemon.jobs = append(daemon.jobs, func(app *appContext) { app.clearTelegram() })
	}
	if clearMatrix {
		daemon.jobs = append(daemon.jobs, func(app *appContext) { app.clearMatrix() })
	}
	if clearPWR {
		daemon.jobs = append(daemon.jobs, func(app *appContext) { app.clearPWRCaptchas() })
	}

	return &daemon
}

func (rt *housekeepingDaemon) run() {
	rt.app.info.Println("Invite daemon started")
	for {
		select {
		case <-rt.ShutdownChannel:
			rt.ShutdownChannel <- "Down"
			return
		case <-time.After(rt.period):
			break
		}
		started := time.Now()

		for _, job := range rt.jobs {
			job(rt.app)
		}

		finished := time.Now()
		duration := finished.Sub(started)
		rt.period = rt.Interval - duration
	}
}

func (rt *housekeepingDaemon) Shutdown() {
	rt.Stopped = true
	rt.ShutdownChannel <- "Down"
	<-rt.ShutdownChannel
	close(rt.ShutdownChannel)
}
