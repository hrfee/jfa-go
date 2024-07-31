package main

import (
	"strconv"
	"time"

	"github.com/hrfee/jfa-go/jellyseerr"
)

func (app *appContext) SynchronizeJellyseerrUser(jfID string) {
	user, imported, err := app.js.GetOrImportUser(jfID)
	if err != nil {
		app.debug.Printf("Failed to get or trigger import for Jellyseerr (user \"%s\"): %v", jfID, err)
		return
	}
	if imported {
		app.debug.Printf("Jellyseerr: Triggered import for Jellyfin user \"%s\" (ID %d)", jfID, user.ID)
	}
	notif, err := app.js.GetNotificationPreferencesByID(user.ID)
	if err != nil {
		app.debug.Printf("Failed to get notification prefs for Jellyseerr (user \"%s\"): %v", jfID, err)
		return
	}

	contactMethods := map[jellyseerr.NotificationsField]any{}
	email, ok := app.storage.GetEmailsKey(jfID)
	if ok && email.Addr != "" && user.Email != email.Addr {
		err = app.js.ModifyMainUserSettings(jfID, jellyseerr.MainUserSettings{Email: email.Addr})
		if err != nil {
			app.err.Printf("Failed to set Jellyseerr email address: %v\n", err)
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
			app.err.Printf("Failed to sync contact methods with Jellyseerr: %v", err)
		}
	}
}

func (app *appContext) SynchronizeJellyseerrUsers() {
	users, status, err := app.jf.GetUsers(false)
	if err != nil || status != 200 {
		app.err.Printf("Failed to get users (%d): %s", status, err)
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
