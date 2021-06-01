package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

func runMigrations(app *appContext) {
	migrateProfiles(app)
	migrateBootstrap(app)
	migrateEmailStorage(app)
	// migrateHyphens(app)
}

// Migrate pre-0.2.0 user templates to profiles
func migrateProfiles(app *appContext) {
	if !(app.storage.policy.BlockedTags == nil && app.storage.configuration.GroupedFolders == nil && len(app.storage.displayprefs) == 0) {
		app.info.Println("Migrating user template files to new profile format")
		app.storage.migrateToProfile()
		for _, path := range [3]string{app.storage.policy_path, app.storage.configuration_path, app.storage.displayprefs_path} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				dir, fname := filepath.Split(path)
				newFname := strings.Replace(fname, ".json", ".old.json", 1)
				err := os.Rename(path, filepath.Join(dir, newFname))
				if err != nil {
					app.err.Fatalf("Failed to rename %s: %s", fname, err)
				}
			}
		}
		app.info.Println("In case of a problem, your original files have been renamed to <file>.old.json")
		app.storage.storeProfiles()
	}
}

// Migrate pre-0.2.5 bootstrap theme choice to a17t version.
func migrateBootstrap(app *appContext) {
	themes := map[string]string{
		"Jellyfin (Dark)": "dark-theme",
		"Default (Light)": "light-theme",
	}

	if app.config.Section("ui").Key("theme").String() == "Bootstrap (Light)" {
		app.config.Section("ui").Key("theme").SetValue("Default (Light)")
	}
	if val, ok := themes[app.config.Section("ui").Key("theme").String()]; ok {
		app.cssClass = val
	}
}

func migrateEmailConfig(app *appContext) {
	tempConfig, _ := ini.Load(app.configPath)
	fmt.Println(warning("Part of your email configuration will be migrated to the new \"messages\" section.\nA backup will be made."))
	err := tempConfig.SaveTo(app.configPath + "_" + commit + ".bak")
	if err != nil {
		app.err.Fatalf("Failed to backup config: %v", err)
		return
	}
	for _, setting := range []string{"use_24h", "date_format", "message"} {
		if val := app.config.Section("email").Key(setting).Value(); val != "" {
			tempConfig.Section("email").Key(setting).SetValue("")
			tempConfig.Section("messages").Key(setting).SetValue(val)
		}
	}
	if app.config.Section("messages").Key("enabled").MustBool(false) || app.config.Section("telegram").Key("enabled").MustBool(false) {
		tempConfig.Section("messages").Key("enabled").SetValue("true")
	}
	err = tempConfig.SaveTo(app.configPath)
	if err != nil {
		app.err.Fatalf("Failed to save config: %v", err)
		return
	}
	app.loadConfig()
}

// Migrate pre-0.3.6 email settings to the new messages section.
// Called just after loading email storage in main.go.
func migrateEmailStorage(app *appContext) error {
	// use_24h was moved to messages, so this checks if migration has already occurred or not.
	if app.config.Section("email").Key("use_24h").Value() == "" {
		return nil
	}
	var emails map[string]interface{}
	err := loadJSON(app.storage.emails_path, &emails)
	if err != nil {
		return err
	}
	newEmails := map[string]EmailAddress{}
	for jfID, addr := range emails {
		newEmails[jfID] = EmailAddress{
			Addr:    addr.(string),
			Contact: true,
		}
	}
	err = storeJSON(app.storage.emails_path+".bak", emails)
	if err != nil {
		return err
	}
	err = storeJSON(app.storage.emails_path, newEmails)
	if err != nil {
		return err
	}
	app.info.Println("Migrated to new email format. A backup has also been made.")
	return nil
}

// Migrate between hyphenated & non-hyphenated user IDs. Doesn't seem to happen anymore, so disabled.
// func migrateHyphens(app *appContext) {
// 	checkVersion := func(version string) int {
// 		numberStrings := strings.Split(version, ".")
// 		n := 0
// 		for _, s := range numberStrings {
// 			num, err := strconv.Atoi(s)
// 			if err == nil {
// 				n += num
// 			}
// 		}
// 		return n
// 	}
// 	if serverType == mediabrowser.JellyfinServer && checkVersion(app.jf.ServerInfo.Version) >= checkVersion("10.7.0") {
// 		// Get users to check if server uses hyphenated userIDs
// 		app.jf.GetUsers(false)
//
// 		noHyphens := true
// 		for id := range app.storage.emails {
// 			if strings.Contains(id, "-") {
// 				noHyphens = false
// 				break
// 			}
// 		}
// 		if noHyphens == app.jf.Hyphens {
// 			var newEmails map[string]interface{}
// 			var newUsers map[string]time.Time
// 			var status, status2 int
// 			var err, err2 error
// 			if app.jf.Hyphens {
// 				app.info.Println(info("Your build of Jellyfin appears to hypenate user IDs. Your emails.json/users.json file will be modified to match."))
// 				time.Sleep(time.Second * time.Duration(3))
// 				newEmails, status, err = app.hyphenateEmailStorage(app.storage.emails)
// 				newUsers, status2, err2 = app.hyphenateUserStorage(app.storage.users)
// 			} else {
// 				app.info.Println(info("Your emails.json/users.json file uses hyphens, but the Jellyfin server no longer does. It will be modified."))
// 				time.Sleep(time.Second * time.Duration(3))
// 				newEmails, status, err = app.deHyphenateEmailStorage(app.storage.emails)
// 				newUsers, status2, err2 = app.deHyphenateUserStorage(app.storage.users)
// 			}
// 			if status != 200 || err != nil {
// 				app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
// 				app.err.Fatalf("Couldn't upgrade emails.json")
// 			}
// 			if status2 != 200 || err2 != nil {
// 				app.err.Printf("Failed to get users from Jellyfin (%d): %v", status, err)
// 				app.err.Fatalf("Couldn't upgrade users.json")
// 			}
// 			emailBakFile := app.storage.emails_path + ".bak"
// 			usersBakFile := app.storage.users_path + ".bak"
// 			err = storeJSON(emailBakFile, app.storage.emails)
// 			err2 = storeJSON(usersBakFile, app.storage.users)
// 			if err != nil {
// 				app.err.Fatalf("couldn't store emails.json backup: %v", err)
// 			}
// 			if err2 != nil {
// 				app.err.Fatalf("couldn't store users.json backup: %v", err)
// 			}
// 			app.storage.emails = newEmails
// 			app.storage.users = newUsers
// 			err = app.storage.storeEmails()
// 			err2 = app.storage.storeUsers()
// 			if err != nil {
// 				app.err.Fatalf("couldn't store emails.json: %v", err)
// 			}
// 			if err2 != nil {
// 				app.err.Fatalf("couldn't store users.json: %v", err)
// 			}
// 		}
// 	}
// }
