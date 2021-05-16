// +build tray

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/emersion/go-autostart"
	"github.com/getlantern/systray"
)

var TRAY = true

func RunTray() {
	systray.Run(onReady, onExit)
}

func onExit() {
	if RUNNING {
		QUIT = true
		RESTART <- true
	}
	os.Remove(SOCK)
}

func onReady() {
	icon, err := localFS.ReadFile("web/favicon.ico")
	if err != nil {
		log.Fatalf("Failed to load favicon: %v", err)
	}
	command := os.Args
	command[0], _ = filepath.Abs(command[0])
	// Make sure to replace any relative paths with absolute ones
	pathArgs := []string{"-d", "-data", "-c", "-config"}
	for i := 1; i < len(command); i++ {
		isPath := false
		for _, p := range pathArgs {
			if command[i-1] == p {
				isPath = true
				break
			}
		}
		if isPath {
			command[i], _ = filepath.Abs(command[i])
		}
	}
	as := &autostart.App{
		Name:        "jfa-go",
		DisplayName: "A user management system for Jellyfin",
		Exec:        command,
	}
	systray.SetIcon(icon)
	systray.SetTitle("jfa-go")
	mStart := systray.AddMenuItem("Start", "Start jfa-go")
	mStop := systray.AddMenuItem("Stop", "Stop jfa-go")
	mRestart := systray.AddMenuItem("Restart", "Restart jfa-go")
	mQuit := systray.AddMenuItem("Quit", "Quit jfa-go")
	mOnLogin := systray.AddMenuItemCheckbox("Run on login", "Run jfa-go on user login.", as.IsEnabled())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		systray.Quit()
		os.Exit(1)
	}()
	defer func() {
		systray.Quit()
	}()

	RESTART = make(chan bool, 1)
	go start(false, true)
	mStart.Disable()
	mStop.Enable()
	mRestart.Enable()
	for {
		select {
		case <-mStart.ClickedCh:
			if !RUNNING {
				go start(false, false)
				mStart.Disable()
				mStop.Enable()
				mRestart.Enable()
			}
		case <-mStop.ClickedCh:
			if RUNNING {
				RESTART <- true
				mStop.Disable()
				mStart.Enable()
				mRestart.Disable()
			}
		case <-mRestart.ClickedCh:
			if RUNNING {
				RESTART <- true
				mStop.Disable()
				mStart.Enable()
				mRestart.Disable()
				for {
					if !RUNNING {
						break
					}
				}
				go start(false, false)
				mStart.Disable()
				mStop.Enable()
				mRestart.Enable()
			}
		case <-mQuit.ClickedCh:
			systray.Quit()
		case <-mOnLogin.ClickedCh:
			fmt.Printf("Checked: %t, Enabled: %t\n", mOnLogin.Checked(), as.IsEnabled())
			if !mOnLogin.Checked() {
				if err := as.Enable(); err != nil {
					log.Printf("Failed to enable autostart on login: %v", err)
				} else {
					mOnLogin.Check()
					log.Printf("Enabled autostart")
				}
			} else {
				if err := as.Disable(); err != nil {
					log.Printf("Failed to disable autostart on login: %v", err)
				} else {
					mOnLogin.Uncheck()
					log.Printf("Disabled autostart")
				}
			}
		}
	}
}
