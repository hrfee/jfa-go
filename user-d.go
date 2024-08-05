package main

import (
	"time"

	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/hrfee/mediabrowser"
	"github.com/lithammer/shortuuid/v3"
)

func newUserDaemon(interval time.Duration, app *appContext) *GenericDaemon {
	d := NewGenericDaemon(interval, app,
		func(app *appContext) {
			app.checkUsers()
		},
	)
	d.Name("User daemon")
	return d
}

func (app *appContext) checkUsers() {
	if len(app.storage.GetUserExpiries()) == 0 {
		return
	}

	app.info.Println(lm.CheckUserExpiries)
	users, status, err := app.jf.GetUsers(false)
	if err != nil || status != 200 {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		return
	}
	mode := "disable"
	if app.config.Section("user_expiry").Key("behaviour").MustString("disable_user") == "delete_user" {
		mode = "delete"
	}

	deleteAfterPeriod := app.config.Section("user_expiry").Key("delete_expired_after_days").MustInt(0)
	if mode == "delete" {
		deleteAfterPeriod = 0
	}

	contact := false
	if messagesEnabled && app.config.Section("user_expiry").Key("send_email").MustBool(true) {
		contact = true
	}
	// Use a map to speed up checking for deleted users later
	userExists := map[string]mediabrowser.User{}
	for _, user := range users {
		userExists[user.ID] = user
	}
	for _, expiry := range app.storage.GetUserExpiries() {
		id := expiry.JellyfinID
		user, ok := userExists[id]
		if !ok {
			app.info.Printf(lm.DeleteExpiryForOldUser, id)
			app.storage.DeleteUserExpiryKey(expiry.JellyfinID)
			continue
		}
		if !time.Now().After(expiry.Expiry) {
			continue
		}
		deleteUserLater := deleteAfterPeriod != 0 && expiry.DeleteAfterPeriod

		// Record activity
		activity := Activity{
			UserID:     id,
			SourceType: ActivityDaemon,
			Time:       time.Now(),
		}

		deleteUserNow := deleteUserLater && time.Now().After(expiry.Expiry.AddDate(0, 0, deleteAfterPeriod))

		if mode == "delete" || deleteUserNow {
			app.info.Printf(lm.DeleteExpiredUser, user.Name)
			deleted := false
			err, deleted = app.DeleteUser(user)
			// Silence unimportant errors
			if deleted {
				err = nil
			}
			activity.Type = ActivityDeletion
			// Store the user name, since there's no longer a user ID to reference back to
			activity.Value = user.Name
		} else if mode == "disable" && !deleteUserLater {
			app.info.Printf(lm.DisableExpiredUser, user.Name)
			// Admins can't be disabled
			// so they're not an admin anymore, sorry
			user.Policy.IsAdministrator = false
			err, _, _ = app.SetUserDisabled(user, true)
			activity.Type = ActivityDisabled
		}
		if !(status == 200 || status == 204) || err != nil {
			app.err.Printf(lm.FailedDeleteOrDisableExpiredUser, user.ID, err)
			continue
		}

		app.storage.SetActivityKey(shortuuid.New(), activity, nil, false)

		// If the we're not gonna be deleting the user later, we don't need this.
		// also, if the admin re-enabled the user, we should get rid of the countdown.
		if deleteAfterPeriod <= 0 || mode == "delete" || deleteUserNow || (deleteUserLater && !user.Policy.IsDisabled) {
			app.storage.DeleteUserExpiryKey(user.ID)
		} else if deleteAfterPeriod > 0 && !deleteUserLater {
			expiry.DeleteAfterPeriod = true
			app.storage.SetUserExpiryKey(user.ID, expiry)
		}
		app.jf.CacheExpiry = time.Now()
		if contact {
			if !ok {
				continue
			}
			name := app.getAddressOrName(user.ID)
			msg, err := app.email.constructUserExpired(app, false)
			if err != nil {
				app.err.Printf(lm.FailedConstructExpiryMessage, user.ID, err)
			} else if err := app.sendByID(msg, user.ID); err != nil {
				app.err.Printf(lm.FailedSendExpiryMessage, user.ID, name, err)
			} else {
				app.err.Printf(lm.SentExpiryMessage, user.ID, name)
			}
		}
	}
}
