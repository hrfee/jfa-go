package main

import (
	"strconv"
	"time"

	"github.com/hrfee/jfa-go/jellyseerr"
	lm "github.com/hrfee/jfa-go/logmessages"
)

func (app *appContext) SynchronizeJellyseerrUser(jfID string) {
	user, imported, err := app.js.GetOrImportUser(jfID)
	if err != nil {
		app.debug.Printf(lm.FailedImportUser, lm.Jellyseerr, jfID, err)
		return
	}
	if imported {
		app.debug.Printf(lm.ImportJellyseerrUser, jfID, user.ID)
	}
	notif, err := app.js.GetNotificationPreferencesByID(user.ID)
	if err != nil {
		app.debug.Printf(lm.FailedGetJellyseerrNotificationPrefs, jfID, err)
		return
	}

	contactMethods := map[jellyseerr.NotificationsField]any{}
	email, ok := app.storage.GetEmailsKey(jfID)
	if ok && email.Addr != "" && user.Email != email.Addr {
		err = app.js.ModifyMainUserSettings(jfID, jellyseerr.MainUserSettings{Email: email.Addr})
		if err != nil {
			app.err.Printf(lm.FailedSetEmailAddress, lm.Jellyseerr, jfID, err)
		} else {
			contactMethods[jellyseerr.FieldEmailEnabled] = email.Contact
		}
	}
	if discordEnabled {
		dcUser, ok := app.storage.GetDiscordKey(jfID)
		if ok && dcUser.ID != "" && notif.DiscordID != dcUser.ID {
			contactMethods[jellyseerr.FieldDiscord] = dcUser.ID
			contactMethods[jellyseerr.FieldDiscordEnabled] = dcUser.Contact
		}
	}
	if telegramEnabled {
		tgUser, ok := app.storage.GetTelegramKey(jfID)
		chatID, _ := strconv.ParseInt(notif.TelegramChatID, 10, 64)
		if ok && tgUser.ChatID != 0 && chatID != tgUser.ChatID {
			u, _ := app.storage.GetTelegramKey(jfID)
			contactMethods[jellyseerr.FieldTelegram] = u.ChatID
			contactMethods[jellyseerr.FieldTelegramEnabled] = tgUser.Contact
		}
	}
	if len(contactMethods) != 0 {
		err := app.js.ModifyNotifications(jfID, contactMethods)
		if err != nil {
			app.err.Printf(lm.FailedSyncContactMethods, lm.Jellyseerr, err)
		}
	}
}

func (app *appContext) SynchronizeJellyseerrUsers() {
	users, err := app.jf.GetUsers(false)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		return
	}
	// I'm sure Jellyseerr can handle it,
	// but past issues with the Jellyfin db scare me from
	// running these concurrently. W/e, its a bg task anyway.
	for _, user := range users {
		app.SynchronizeJellyseerrUser(user.ID)
	}
}

func newJellyseerrDaemon(interval time.Duration, app *appContext) *GenericDaemon {
	d := NewGenericDaemon(interval, app,
		func(app *appContext) {
			app.SynchronizeJellyseerrUsers()
		},
	)
	d.Name("Jellyseerr import daemon")
	return d
}
