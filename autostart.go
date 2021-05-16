// +build tray

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/emersion/go-autostart"
	"github.com/getlantern/systray"
)

type Autostart struct {
	as       *autostart.App
	enabled  bool
	menuitem *systray.MenuItem
	clicked  chan bool
}

func NewAutostart(name, displayname, trayName, trayTooltip string) *Autostart {
	a := &Autostart{
		as: &autostart.App{
			Name:        name,
			DisplayName: displayname,
		},
		enabled: true,
		clicked: make(chan bool),
	}
	a.menuitem = systray.AddMenuItemCheckbox(trayName, trayTooltip, a.as.IsEnabled())
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
	a.as.Exec = command
	return a
}

func (a *Autostart) HandleCheck() {
	for range a.menuitem.ClickedCh {
		if !a.menuitem.Checked() {
			if err := a.as.Enable(); err != nil {
				log.Printf("Failed to enable autostart on login: %v", err)
			} else {
				a.menuitem.Check()
				log.Printf("Enabled autostart")
			}
		} else {
			if err := a.as.Disable(); err != nil {
				log.Printf("Failed to disable autostart on login: %v", err)
			} else {
				a.menuitem.Uncheck()
				log.Printf("Disabled autostart")
			}
		}
	}
}
