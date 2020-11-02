package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (app *appContext) StartPWR() {
	app.info.Println("Starting password reset daemon")
	path := app.config.Section("password_resets").Key("watch_directory").String()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		app.err.Printf("Failed to start password reset daemon: Directory \"%s\" doesn't exist", path)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		app.err.Printf("Couldn't initialise password reset daemon")
		return
	}
	defer watcher.Close()

	done := make(chan bool)
	go pwrMonitor(app, watcher)
	err = watcher.Add(path)
	if err != nil {
		app.err.Printf("Failed to start password reset daemon: %s", err)
	}
	<-done
}

type Pwr struct {
	Pin      string    `json:"Pin"`
	Username string    `json:"UserName"`
	Expiry   time.Time `json:"ExpirationDate"`
}

func pwrMonitor(app *appContext, watcher *fsnotify.Watcher) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write && strings.Contains(event.Name, "passwordreset") {
				var pwr Pwr
				data, err := ioutil.ReadFile(event.Name)
				if err != nil {
					return
				}
				err = json.Unmarshal(data, &pwr)
				if len(pwr.Pin) == 0 || err != nil {
					return
				}
				app.info.Printf("New password reset for user \"%s\"", pwr.Username)
				if currentTime := time.Now(); pwr.Expiry.After(currentTime) {
					user, status, err := app.jf.UserByName(pwr.Username, false)
					if !(status == 200 || status == 204) || err != nil {
						app.err.Printf("Failed to get users from Jellyfin: Code %d", status)
						app.debug.Printf("Error: %s", err)
						return
					}
					app.storage.loadEmails()
					var address string
					uid := user["Id"]
					if uid == nil {
						app.err.Printf("Couldn't get user ID for user \"%s\"", pwr.Username)
						app.debug.Printf("user maplength: %d", len(user))
						return
					}
					addr, ok := app.storage.emails[user["Id"].(string)]
					if !ok || addr == nil {
						app.err.Printf("Couldn't find email for user \"%s\". Make sure it's set", pwr.Username)
						return
					}
					address = addr.(string)
					msg, err := app.email.constructReset(pwr, app)
					if err != nil {
						app.err.Printf("Failed to construct password reset email for %s", pwr.Username)
						app.debug.Printf("%s: Error: %s", pwr.Username, err)
					} else if err := app.email.send(address, msg); err != nil {
						app.err.Printf("Failed to send password reset email to \"%s\"", address)
						app.debug.Printf("%s: Error: %s", pwr.Username, err)
					} else {
						app.info.Printf("Sent password reset email to \"%s\"", address)
					}
				} else {
					app.err.Printf("Password reset for user \"%s\" has already expired (%s). Check your time settings.", pwr.Username, pwr.Expiry)
				}

			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			app.err.Printf("Password reset daemon: %s", err)
		}
	}
}
