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
				if ct := time.Now(); pwr.Expiry.After(ct) {
					user, status, err := app.jf.userByName(pwr.Username, false)
					if !(status == 200 || status == 204) || err != nil {
						app.err.Printf("Failed to get users from Jellyfin: Code %d", status)
						app.debug.Printf("Error: %s", err)
						return
					}
					app.storage.loadEmails()
					address, ok := app.storage.emails[user["Id"].(string)].(string)
					if !ok {
						app.err.Printf("Couldn't find email for user \"%s\". Make sure it's set", pwr.Username)
						return
					}
					if app.email.constructReset(pwr, app) != nil {
						app.err.Printf("Failed to construct password reset email for %s", pwr.Username)
					} else if app.email.send(address, app) != nil {
						app.err.Printf("Failed to send password reset email to \"%s\"", address)
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
