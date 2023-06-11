//go:build tray
// +build tray

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getlantern/systray"
)

var TRAY = true

func RunTray() {
	systray.Run(onReady, onExit)
}

func QuitTray() {
	systray.Quit()
}

func onExit() {
	if RUNNING {
		QUIT = true
		RESTART <- true
		// Safety Sleep (Ensure shutdown tasks get done)
		time.Sleep(time.Second)
	}
	os.Remove(SOCK)
}

func onReady() {
	icon, err := localFS.ReadFile("web/favicon.ico")
	if err != nil {
		log.Fatalf("Failed to load favicon: %v", err)
	}
	systray.SetIcon(icon)
	systray.SetTitle("jfa-go")
	mStart := systray.AddMenuItem("Start", "Start jfa-go")
	mStop := systray.AddMenuItem("Stop", "Stop jfa-go")
	mRestart := systray.AddMenuItem("Restart", "Restart jfa-go")
	mOpenLogs := systray.AddMenuItem("Open logs", "Open jfa-go log file.")
	as := NewAutostart("jfa-go", "A user management system for Jellyfin", "Run on login", "Run jfa-go on user login.")
	mQuit := systray.AddMenuItem("Quit", "Quit jfa-go")

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
	TRAYRESTART = make(chan bool, 1)
	go start(false, true)
	mStart.Disable()
	mStop.Enable()
	mRestart.Enable()
	go as.HandleCheck()
	trayRestart := func() {
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
	}
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
		case <-TRAYRESTART:
			trayRestart()
		case <-mRestart.ClickedCh:
			trayRestart()
		case <-mOpenLogs.ClickedCh:
			log.Printf("Opening %s\n", logPath)
			OpenFile(logPath)
		case <-mQuit.ClickedCh:
			systray.Quit()
		}
	}
}
