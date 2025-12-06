package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/hrfee/jfa-go/jellyseerr"
	lm "github.com/hrfee/jfa-go/logmessages"
)

type JellyseerrInitialSyncStatus struct {
	Done bool
}

// Ensure the Jellyseerr cache is up to date before calling.
func (app *appContext) SynchronizeJellyseerrUser(jfID string) {
	user, imported, err := app.js.GetOrImportUser(jfID, true)
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
			if strings.Contains(err.Error(), "INVALID_EMAIL") {
				app.err.Printf(lm.FailedSetEmailAddress, lm.Jellyseerr, jfID, err.Error()+"\""+email.Addr+"\"")
			} else {
				app.err.Printf(lm.FailedSetEmailAddress, lm.Jellyseerr, jfID, err)
			}
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
			contactMethods[jellyseerr.FieldTelegram] = strconv.FormatInt(u.ChatID, 10)
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
	jsSync := JellyseerrInitialSyncStatus{}
	app.storage.db.Get("jellyseerr_inital_sync_status", &jsSync)
	if jsSync.Done {
		return
	}

	users, err := app.jf.GetUsers(false)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		return
	}
	app.js.ReloadCache()
	// I'm sure Jellyseerr can handle it,
	// but past issues with the Jellyfin db scare me from
	// running these concurrently. W/e, its a bg task anyway.
	for _, user := range users {
		app.SynchronizeJellyseerrUser(user.ID)
	}
	// Don't run again until this flag is unset
	// Stored in the DB as it's not something the user needs to see.
	app.storage.db.Upsert("jellyseerr_inital_sync_status", JellyseerrInitialSyncStatus{true})
}

// Not really a normal daemon, since it'll only fire once when the feature is enabled.
func newJellyseerrDaemon(interval time.Duration, app *appContext) *GenericDaemon {
	d := NewGenericDaemon(interval, app,
		func(app *appContext) {
			app.SynchronizeJellyseerrUsers()
		},
	)
	d.Name("Jellyseerr import")

	jsSync := JellyseerrInitialSyncStatus{}
	app.storage.db.Get("jellyseerr_inital_sync_status", &jsSync)
	if jsSync.Done {
		return nil
	}

	return d
}
