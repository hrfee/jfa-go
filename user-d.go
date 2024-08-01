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
	phrase := lm.DisableExpiredUser
	if app.config.Section("user_expiry").Key("behaviour").MustString("disable_user") == "delete_user" {
		mode = "delete"
		phrase = lm.DeleteExpiredUser
	}
	contact := false
	if messagesEnabled && app.config.Section("user_expiry").Key("send_email").MustBool(true) {
		contact = true
	}
	// Use a map to speed up checking for deleted users later
	userExists := map[string]bool{}
	for _, user := range users {
		userExists[user.ID] = true
	}
	for _, expiry := range app.storage.GetUserExpiries() {
		id := expiry.JellyfinID
		if _, ok := userExists[id]; !ok {
			app.info.Printf(lm.DeleteExpiryForOldUser, id)
			app.storage.DeleteUserExpiryKey(expiry.JellyfinID)
		} else if time.Now().After(expiry.Expiry) {
			found := false
			var user mediabrowser.User
			for _, u := range users {
				if u.ID == id {
					found = true
					user = u
					break
				}
			}
			if !found {
				app.storage.DeleteUserExpiryKey(expiry.JellyfinID)
				continue
			}
			app.info.Printf(phrase, user.Name)

			// Record activity
			activity := Activity{
				UserID:     id,
				SourceType: ActivityDaemon,
				Time:       time.Now(),
			}

			if mode == "delete" {
				status, err = app.jf.DeleteUser(id)
				activity.Type = ActivityDeletion
				activity.Value = user.Name
			} else if mode == "disable" {
				user.Policy.IsDisabled = true
				// Admins can't be disabled
				user.Policy.IsAdministrator = false
				status, err = app.jf.SetPolicy(id, user.Policy)
				activity.Type = ActivityDisabled
			}
			if !(status == 200 || status == 204) || err != nil {
				app.err.Printf(lm.FailedDeleteOrDisableExpiredUser, user.ID, err)
				continue
			}

			app.storage.SetActivityKey(shortuuid.New(), activity, nil, false)

			app.storage.DeleteUserExpiryKey(expiry.JellyfinID)
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
					app.err.Printf(lm.FailedConstructExpiryMessage, user.ID, name, err)
				} else {
					app.err.Printf(lm.SentExpiryMessage, user.ID, name)
				}
			}
		}
	}
}
