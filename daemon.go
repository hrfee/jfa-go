package main

import "time"

// clearEmails removes stored emails for users which no longer exist.
// meant to be called with other such housekeeping functions, so assumes
// the user cache is fresh.
func (app *appContext) clearEmails() {
	app.debug.Println("Housekeeping: removing unused email addresses")
	users, status, err := app.jf.GetUsers(false)
	if status != 200 || err != nil || len(users) == 0 {
		app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
		return
	}
	// Rebuild email storage to from existing users to reduce time complexity
	emails := emailStore{}
	app.storage.emailsLock.Lock()
	for _, user := range users {
		if email, ok := app.storage.GetEmailsKey(user.ID); ok {
			emails[user.ID] = email
		}
	}
	app.storage.emails = emails
	app.storage.storeEmails()
	app.storage.emailsLock.Unlock()
}

// clearDiscord does the same as clearEmails, but for Discord Users.
func (app *appContext) clearDiscord() {
	app.debug.Println("Housekeeping: removing unused Discord IDs")
	users, status, err := app.jf.GetUsers(false)
	if status != 200 || err != nil || len(users) == 0 {
		app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
		return
	}
	// Rebuild discord storage to from existing users to reduce time complexity
	dcUsers := discordStore{}
	app.storage.discordLock.Lock()
	for _, user := range users {
		if dcUser, ok := app.storage.GetDiscordKey(user.ID); ok {
			dcUsers[user.ID] = dcUser
		}
	}
	app.storage.discord = dcUsers
	app.storage.storeDiscordUsers()
	app.storage.discordLock.Unlock()
}

// clearMatrix does the same as clearEmails, but for Matrix Users.
func (app *appContext) clearMatrix() {
	app.debug.Println("Housekeeping: removing unused Matrix IDs")
	users, status, err := app.jf.GetUsers(false)
	if status != 200 || err != nil || len(users) == 0 {
		app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
		return
	}
	// Rebuild matrix storage to from existing users to reduce time complexity
	mxUsers := matrixStore{}
	app.storage.matrixLock.Lock()
	for _, user := range users {
		if mxUser, ok := app.storage.GetMatrixKey(user.ID); ok {
			mxUsers[user.ID] = mxUser
		}
	}
	app.storage.matrix = mxUsers
	app.storage.storeMatrixUsers()
	app.storage.matrixLock.Unlock()
}

// clearTelegram does the same as clearEmails, but for Telegram Users.
func (app *appContext) clearTelegram() {
	app.debug.Println("Housekeeping: removing unused Telegram IDs")
	users, status, err := app.jf.GetUsers(false)
	if status != 200 || err != nil || len(users) == 0 {
		app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
		return
	}
	// Rebuild telegram storage to from existing users to reduce time complexity
	tgUsers := telegramStore{}
	app.storage.telegramLock.Lock()
	for _, user := range users {
		if tgUser, ok := app.storage.GetTelegramKey(user.ID); ok {
			tgUsers[user.ID] = tgUser
		}
	}
	app.storage.telegram = tgUsers
	app.storage.storeTelegramUsers()
	app.storage.telegramLock.Unlock()
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
		rt.app.storage.loadInvites()

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
