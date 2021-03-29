package main

import (
	"time"

	"github.com/hrfee/mediabrowser"
)

type userDaemon struct {
	Stopped         bool
	ShutdownChannel chan string
	Interval        time.Duration
	period          time.Duration
	app             *appContext
}

func newUserDaemon(interval time.Duration, app *appContext) *userDaemon {
	return &userDaemon{
		Stopped:         false,
		ShutdownChannel: make(chan string),
		Interval:        interval,
		period:          interval,
		app:             app,
	}
}

func (rt *userDaemon) run() {
	rt.app.info.Println("User daemon started")
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
		rt.app.checkUsers()
		finished := time.Now()
		duration := finished.Sub(started)
		rt.period = rt.Interval - duration
	}
}

func (app *appContext) checkUsers() {
	if len(app.storage.users) == 0 {
		return
	}
	app.info.Println("Daemon: Checking for user expiry")
	users, status, err := app.jf.GetUsers(false)
	if err != nil || status != 200 {
		app.err.Printf("Failed to get users (%d): %s", status, err)
		return
	}
	mode := "disable"
	termPlural := "Disabling"
	if app.config.Section("user_expiry").Key("behaviour").MustString("disable_user") == "delete_user" {
		mode = "delete"
		termPlural = "Deleting"
	}
	email := false
	if emailEnabled && app.config.Section("user_expiry").Key("send_email").MustBool(true) {
		email = true
	}
	for id, expiry := range app.storage.users {
		if time.Now().After(expiry) {
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
				app.info.Printf("Expired user already deleted, ignoring.")
				delete(app.storage.users, id)
				continue
			}
			app.info.Printf("%s expired user \"%s\"", termPlural, user.Name)
			if mode == "delete" {
				status, err = app.jf.DeleteUser(id)
			} else if mode == "disable" {
				user.Policy.IsDisabled = true
				status, err = app.jf.SetPolicy(id, user.Policy)
			}
			if !(status == 200 || status == 204) || err != nil {
				app.err.Printf("Failed to %s \"%s\" (%d): %s", mode, user.Name, status, err)
				continue
			}
			delete(app.storage.users, id)
			if email {
				address, ok := app.storage.emails[id]
				if !ok {
					continue
				}
				msg, err := app.email.constructUserExpired(app, false)
				if err != nil {
					app.err.Printf("Failed to construct expiry email for \"%s\": %s", user.Name, err)
				} else if err := app.email.send(msg, address.(string)); err != nil {
					app.err.Printf("Failed to send expiry email to \"%s\": %s", user.Name, err)
				} else {
					app.info.Printf("Sent expiry notification to \"%s\"", address.(string))
				}
			}
		}
	}
	err = app.storage.storeUsers()
	if err != nil {
		app.err.Printf("Failed to store user duration: %s", err)
	}
}
