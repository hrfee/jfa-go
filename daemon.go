package main

import "time"

// clearEmails removes stored emails for users which no longer exist.
// meant to be called with other such housekeeping functions, so assumes
// the user cache is fresh.
func (app *appContext) clearEmails() {
	app.debug.Println("Housekeeping: removing unused email addresses")
	emails := app.storage.GetEmails()
	for _, email := range emails {
		_, status, err := app.jf.UserByID(email.JellyfinID, false)
		if status == 200 && err == nil {
			continue
		}
		app.storage.DeleteEmailsKey(email.JellyfinID)
	}
}

// clearDiscord does the same as clearEmails, but for Discord Users.
func (app *appContext) clearDiscord() {
	app.debug.Println("Housekeeping: removing unused Discord IDs")
	discordUsers := app.storage.GetDiscord()
	for _, discordUser := range discordUsers {
		_, status, err := app.jf.UserByID(discordUser.JellyfinID, false)
		if status == 200 && err == nil {
			continue
		}
		app.storage.DeleteDiscordKey(discordUser.JellyfinID)
	}
}

// clearMatrix does the same as clearEmails, but for Matrix Users.
func (app *appContext) clearMatrix() {
	app.debug.Println("Housekeeping: removing unused Matrix IDs")
	matrixUsers := app.storage.GetMatrix()
	for _, matrixUser := range matrixUsers {
		_, status, err := app.jf.UserByID(matrixUser.JellyfinID, false)
		if status == 200 && err == nil {
			continue
		}
		app.storage.DeleteMatrixKey(matrixUser.JellyfinID)
	}
}

// clearTelegram does the same as clearEmails, but for Telegram Users.
func (app *appContext) clearTelegram() {
	app.debug.Println("Housekeeping: removing unused Telegram IDs")
	telegramUsers := app.storage.GetTelegram()
	for _, telegramUser := range telegramUsers {
		_, status, err := app.jf.UserByID(telegramUser.JellyfinID, false)
		if status == 200 && err == nil {
			continue
		}
		app.storage.DeleteTelegramKey(telegramUser.JellyfinID)
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
	daemon.jobs = []func(app *appContext){func(app *appContext) {
		app.debug.Println("Housekeeping: Checking for expired invites")
		app.checkInvites()
	}}

	clearEmail := app.config.Section("email").Key("require_unique").MustBool(false)
	clearDiscord := app.config.Section("discord").Key("require_unique").MustBool(false)
	clearTelegram := app.config.Section("telegram").Key("require_unique").MustBool(false)
	clearMatrix := app.config.Section("matrix").Key("require_unique").MustBool(false)

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
