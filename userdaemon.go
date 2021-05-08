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
		rt.app.checkUsers()
		finished := time.Now()
		duration := finished.Sub(started)
		rt.period = rt.Interval - duration
	}
}

func (rt *userDaemon) shutdown() {
	rt.Stopped = true
	rt.ShutdownChannel <- "Down"
	<-rt.ShutdownChannel
	close(rt.ShutdownChannel)
}

func (app *appContext) checkUsers() {
	if err := app.storage.loadUsers(); err != nil {
		app.err.Printf("Failed to load user expiries: %v", err)
		return
	}
	app.storage.usersLock.Lock()
	defer app.storage.usersLock.Unlock()
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
	contact := false
	if messagesEnabled && app.config.Section("user_expiry").Key("send_email").MustBool(true) {
		contact = true
	}
	// Use a map to speed up checking for deleted users later
	userExists := map[string]bool{}
	for _, user := range users {
		userExists[user.ID] = true
	}
	for id, expiry := range app.storage.users {
		if _, ok := userExists[id]; !ok {
			app.info.Printf("Deleting expiry for non-existent user \"%s\"", id)
			delete(app.storage.users, id)
		} else if time.Now().After(expiry) {
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
				// Admins can't be disabled
				user.Policy.IsAdministrator = false
				status, err = app.jf.SetPolicy(id, user.Policy)
			}
			if !(status == 200 || status == 204) || err != nil {
				app.err.Printf("Failed to %s \"%s\" (%d): %s", mode, user.Name, status, err)
				continue
			}
			delete(app.storage.users, id)
			app.jf.CacheExpiry = time.Now()
			if contact {
				if !ok {
					continue
				}
				name := app.getAddressOrName(user.ID)
				msg, err := app.email.constructUserExpired(app, false)
				if err != nil {
					app.err.Printf("Failed to construct expiry message for \"%s\": %s", user.Name, err)
				} else if err := app.sendByID(msg, user.ID); err != nil {
					app.err.Printf("Failed to send expiry message to \"%s\": %s", name, err)
				} else {
					app.info.Printf("Sent expiry notification to \"%s\"", name)
				}
			}
		}
	}
	err = app.storage.storeUsers()
	if err != nil {
		app.err.Printf("Failed to store user expiries: %s", err)
	}
}
