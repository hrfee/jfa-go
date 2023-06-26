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
	migrateNotificationMethods(app)
	linkExistingOmbiDiscordTelegram(app)
	// migrateHyphens(app)
	migrateToBadger(app)
}

// Migrate pre-0.2.0 user templates to profiles
func migrateProfiles(app *appContext) {
	if app.storage.deprecatedPolicy.BlockedTags == nil && app.storage.deprecatedConfiguration.GroupedFolders == nil && len(app.storage.deprecatedDisplayprefs) == 0 {
		return
	}
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

// Migrate pre-0.2.5 bootstrap theme choice to a17t version.
func migrateBootstrap(app *appContext) {
	themes := map[string]string{
		"Jellyfin (Dark)": "dark",
		"Default (Light)": "light",
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
		switch addr.(type) {
		case string:
			newEmails[jfID] = EmailAddress{
				Addr:    addr.(string),
				Contact: true,
			}
		// In case email settings still persist after migration has already happened
		case map[string]interface{}:
			return nil
		default:
			return fmt.Errorf("email address was type %T, not string: \"%+v\"\n", addr, addr)
		}
	}
	config, err := ini.Load(app.configPath)
	if err != nil {
		return err
	}
	config.Section("email").Key("use_24h").SetValue("")
	if err := config.SaveTo(app.configPath); err != nil {
		return err
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

// Pre-0.4.0, Admin notifications for invites were indexed by and only sent to email addresses. Now, when Jellyfin Login is enabled, They are indexed by the admin's Jellyfin ID, and send by any method enabled for them. This migrates storage to that format.
func migrateNotificationMethods(app *appContext) error {
	if !app.config.Section("ui").Key("jellyfin_login").MustBool(false) {
		return nil
	}
	changes := false
	for code, invite := range app.storage.deprecatedInvites {
		if invite.Notify == nil {
			continue
		}
		for address, notifyPrefs := range invite.Notify {
			if !strings.Contains(address, "@") {
				continue
			}
			for _, email := range app.storage.GetEmails() {
				if email.Addr == address {
					invite.Notify[email.JellyfinID] = notifyPrefs
					delete(invite.Notify, address)
					changes = true
					break
				}
			}
		}
		if changes {
			app.storage.deprecatedInvites[code] = invite
		}
	}
	if changes {
		app.info.Printf("Migrated to modified invite storage format.")
		return app.storage.storeInvites()
	}
	return nil
}

// Pre-0.4.0, Ombi users were created without linking their Discord & Telegram accounts. This will add them.
func linkExistingOmbiDiscordTelegram(app *appContext) error {
	if !discordEnabled && !telegramEnabled {
		return nil
	}
	if !app.config.Section("ombi").Key("enabled").MustBool(false) {
		return nil
	}
	idList := map[string][2]string{}
	for _, user := range app.storage.GetDiscord() {
		idList[user.JellyfinID] = [2]string{user.ID, ""}
	}
	for _, user := range app.storage.GetTelegram() {
		vals, ok := idList[user.JellyfinID]
		if !ok {
			vals = [2]string{"", ""}
		}
		vals[1] = user.Username
		idList[user.JellyfinID] = vals
	}
	for jfID, ids := range idList {
		ombiUser, status, err := app.getOmbiUser(jfID)
		if status != 200 || err != nil {
			app.debug.Printf("Failed to get Ombi user with Discord/Telegram \"%s\"/\"%s\" (%d): %v", ids[0], ids[1], status, err)
			continue
		}
		_, status, err = app.ombi.SetNotificationPrefs(ombiUser, ids[0], ids[1])
		if status != 200 || err != nil {
			app.debug.Printf("Failed to set prefs for Ombi user \"%s\" (%d): %v", ombiUser["userName"].(string), status, err)
			continue
		}
	}
	return nil
}

// MigrationStatus is just used to store whether data from JSON files has been migrated to the DB.
type MigrationStatus struct {
	Done bool
}

func loadLegacyData(app *appContext) {
	app.storage.invite_path = app.config.Section("files").Key("invites").String()
	if err := app.storage.loadInvites(); err != nil {
		app.err.Printf("LegacyData: Failed to load Invites: %v", err)
	}
	app.storage.emails_path = app.config.Section("files").Key("emails").String()
	if err := app.storage.loadEmails(); err != nil {
		app.err.Printf("LegacyData: Failed to load Emails: %v", err)
		err := migrateEmailStorage(app)
		if err != nil {
			app.err.Printf("LegacyData: Failed to migrate Email storage: %v", err)
		}
	}
	app.storage.users_path = app.config.Section("files").Key("users").String()
	if err := app.storage.loadUserExpiries(); err != nil {
		app.err.Printf("LegacyData: Failed to load Users: %v", err)
	}
	app.storage.telegram_path = app.config.Section("files").Key("telegram_users").String()
	if err := app.storage.loadTelegramUsers(); err != nil {
		app.err.Printf("LegacyData: Failed to load Telegram users: %v", err)
	}
	app.storage.discord_path = app.config.Section("files").Key("discord_users").String()
	if err := app.storage.loadDiscordUsers(); err != nil {
		app.err.Printf("LegacyData: Failed to load Discord users: %v", err)
	}
	app.storage.matrix_path = app.config.Section("files").Key("matrix_users").String()
	if err := app.storage.loadMatrixUsers(); err != nil {
		app.err.Printf("LegacyData: Failed to load Matrix users: %v", err)
	}
	app.storage.announcements_path = app.config.Section("files").Key("announcements").String()
	if err := app.storage.loadAnnouncements(); err != nil {
		app.err.Printf("LegacyData: Failed to load announcement templates: %v", err)
	}

	app.storage.profiles_path = app.config.Section("files").Key("user_profiles").String()
	app.storage.loadProfiles()

	app.storage.customEmails_path = app.config.Section("files").Key("custom_emails").String()
	app.storage.loadCustomEmails()

	app.MustSetValue("user_page", "enabled", "true")
	if app.config.Section("user_page").Key("enabled").MustBool(false) {
		app.storage.userPage_path = app.config.Section("files").Key("custom_user_page_content").String()
		app.storage.loadUserPageContent()
	}
}

func migrateToBadger(app *appContext) {
	// Check the DB to see if we've already migrated
	migrated := MigrationStatus{}
	app.storage.db.Get("migrated_to_db", &migrated)
	if migrated.Done {
		return
	}
	app.info.Println("Migrating to Badger(hold)")
	loadLegacyData(app)
	for k, v := range app.storage.deprecatedAnnouncements {
		app.storage.SetAnnouncementsKey(k, v)
	}

	for jfID, v := range app.storage.deprecatedDiscord {
		app.storage.SetDiscordKey(jfID, v)
	}

	for jfID, v := range app.storage.deprecatedTelegram {
		app.storage.SetTelegramKey(jfID, v)
	}

	for jfID, v := range app.storage.deprecatedMatrix {
		app.storage.SetMatrixKey(jfID, v)
	}

	for jfID, v := range app.storage.deprecatedEmails {
		app.storage.SetEmailsKey(jfID, v)
	}

	for k, v := range app.storage.deprecatedInvites {
		app.storage.SetInvitesKey(k, v)
	}

	for k, v := range app.storage.deprecatedUserExpiries {
		app.storage.SetUserExpiryKey(k, UserExpiry{Expiry: v})
	}

	for k, v := range app.storage.deprecatedProfiles {
		if v.Configuration.GroupedFolders != nil || len(v.Displayprefs) != 0 {
			v.Homescreen = true
		}
		app.storage.SetProfileKey(k, v)
	}

	if _, ok := app.storage.GetCustomContentKey("UserCreated"); !ok {
		app.storage.SetCustomContentKey("UserCreated", app.storage.deprecatedCustomEmails.UserCreated)
	}
	if _, ok := app.storage.GetCustomContentKey("InviteExpiry"); !ok {
		app.storage.SetCustomContentKey("InviteExpiry", app.storage.deprecatedCustomEmails.InviteExpiry)
	}
	if _, ok := app.storage.GetCustomContentKey("PasswordReset"); !ok {
		app.storage.SetCustomContentKey("PasswordReset", app.storage.deprecatedCustomEmails.PasswordReset)
	}
	if _, ok := app.storage.GetCustomContentKey("UserDeleted"); !ok {
		app.storage.SetCustomContentKey("UserDeleted", app.storage.deprecatedCustomEmails.UserDeleted)
	}
	if _, ok := app.storage.GetCustomContentKey("UserDisabled"); !ok {
		app.storage.SetCustomContentKey("UserDisabled", app.storage.deprecatedCustomEmails.UserDisabled)
	}
	if _, ok := app.storage.GetCustomContentKey("UserEnabled"); !ok {
		app.storage.SetCustomContentKey("UserEnabled", app.storage.deprecatedCustomEmails.UserEnabled)
	}
	if _, ok := app.storage.GetCustomContentKey("InviteEmail"); !ok {
		app.storage.SetCustomContentKey("InviteEmail", app.storage.deprecatedCustomEmails.InviteEmail)
	}
	if _, ok := app.storage.GetCustomContentKey("WelcomeEmail"); !ok {
		app.storage.SetCustomContentKey("WelcomeEmail", app.storage.deprecatedCustomEmails.WelcomeEmail)
	}
	if _, ok := app.storage.GetCustomContentKey("EmailConfirmation"); !ok {
		app.storage.SetCustomContentKey("EmailConfirmation", app.storage.deprecatedCustomEmails.EmailConfirmation)
	}
	if _, ok := app.storage.GetCustomContentKey("UserExpired"); !ok {
		app.storage.SetCustomContentKey("UserExpired", app.storage.deprecatedCustomEmails.UserExpired)
	}
	if _, ok := app.storage.GetCustomContentKey("UserLogin"); !ok {
		app.storage.SetCustomContentKey("UserLogin", app.storage.deprecatedUserPageContent.Login)
	}
	if _, ok := app.storage.GetCustomContentKey("UserPage"); !ok {
		app.storage.SetCustomContentKey("UserPage", app.storage.deprecatedUserPageContent.Page)
	}

	err := app.storage.db.Upsert("migrated_to_db", MigrationStatus{true})
	if err != nil {
		app.err.Fatalf("Failed to migrate to DB: %v\n", err)
	}
	app.info.Println("All data migrated to database. JSON files in the config folder can be deleted if you are sure all data is correct in the app. Create an issue if you have problems.")
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
// 		for _, e := range app.storage.GetEmails() {
// 			if strings.Contains(e.JellyfinID, "-") {
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
// 			err2 = app.storage.storeUserExpiries()
// 			if err != nil {
// 				app.err.Fatalf("couldn't store emails.json: %v", err)
// 			}
// 			if err2 != nil {
// 				app.err.Fatalf("couldn't store users.json: %v", err)
// 			}
// 		}
// 	}
// }
