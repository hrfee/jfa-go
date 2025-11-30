package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/common"
	lm "github.com/hrfee/jfa-go/logmessages"
	ombiLib "github.com/hrfee/jfa-go/ombi"
	"github.com/hrfee/mediabrowser"
)

func (app *appContext) getOmbiUser(jfID string) (map[string]interface{}, error) {
	jfUser, err := app.jf.UserByID(jfID, false)
	if err != nil {
		return nil, err
	}
	username := jfUser.Name
	email := ""
	if e, ok := app.storage.GetEmailsKey(jfID); ok {
		email = e.Addr
	}
	user, err := app.ombi.getUser(username, email)
	return user, err
}

func (ombi *OmbiWrapper) getUser(username string, email string) (map[string]interface{}, error) {
	ombiUsers, err := ombi.GetUsers()
	if err != nil {
		return nil, err
	}
	for _, ombiUser := range ombiUsers {
		ombiAddr := ""
		if a, ok := ombiUser["emailAddress"]; ok && a != nil {
			ombiAddr = a.(string)
		}
		if ombiUser["userName"].(string) == username || (ombiAddr == email && email != "") {
			return ombiUser, err
		}
	}
	// Gets a generic "not found" type error
	return nil, common.GenericErr(404, err)
}

// Returns a user with the given name who has been imported from Jellyfin/Emby by Ombi
func (ombi *OmbiWrapper) getImportedUser(name string) (map[string]interface{}, error) {
	// Ombi User Types: 3/4 = Emby, 5 = Jellyfin
	ombiUsers, err := ombi.GetUsers()
	if err != nil {
		return nil, err
	}
	for _, ombiUser := range ombiUsers {
		if ombiUser["userName"].(string) == name {
			uType, ok := ombiUser["userType"].(int)
			if !ok { // Don't know if Ombi somehow allows duplicate usernames
				continue
			}
			if serverType == mediabrowser.JellyfinServer && uType != 5 { // Jellyfin
				continue
			} else if uType != 3 && uType != 4 { // Emby
				continue
			}
			return ombiUser, err
		}
	}
	// Gets a generic "not found" type error
	return nil, common.GenericErr(404, err)
}

// @Summary Get a list of Ombi users.
// @Produce json
// @Success 200 {object} ombiUsersDTO
// @Failure 500 {object} stringResponse
// @Router /ombi/users [get]
// @Security Bearer
// @tags Ombi
func (app *appContext) OmbiUsers(gc *gin.Context) {
	users, err := app.ombi.GetUsers()
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Ombi, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	userlist := make([]ombiUser, len(users))
	for i, data := range users {
		userlist[i] = ombiUser{
			Name: data["userName"].(string),
			ID:   data["id"].(string),
		}
	}
	gc.JSON(200, ombiUsersDTO{Users: userlist})
}

// @Summary Store Ombi user template in an existing profile.
// @Produce json
// @Param ombiUser body ombiUser true "User to source settings from"
// @Param profile path string true "Name of profile to store in"
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /profiles/ombi/{profile} [post]
// @Security Bearer
// @tags Ombi
func (app *appContext) SetOmbiProfile(gc *gin.Context) {
	var req ombiUser
	gc.BindJSON(&req)
	escapedProfileName := gc.Param("profile")
	profileName, _ := url.QueryUnescape(escapedProfileName)
	profile, ok := app.storage.GetProfileKey(profileName)
	if !ok {
		respondBool(400, false, gc)
		return
	}
	template, err := app.ombi.TemplateByID(req.ID)
	if err != nil || len(template) == 0 {
		app.err.Printf(lm.FailedGetUsers, lm.Ombi, err)
		respond(500, "Couldn't get user", gc)
		return
	}
	profile.Ombi = template
	app.storage.SetProfileKey(profileName, profile)
	respondBool(204, true, gc)
}

// @Summary Remove ombi user template from a profile.
// @Produce json
// @Param profile path string true "Name of profile to store in"
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /profiles/ombi/{profile} [delete]
// @Security Bearer
// @tags Ombi
func (app *appContext) DeleteOmbiProfile(gc *gin.Context) {
	escapedProfileName := gc.Param("profile")
	profileName, _ := url.QueryUnescape(escapedProfileName)
	profile, ok := app.storage.GetProfileKey(profileName)
	if !ok {
		respondBool(400, false, gc)
		return
	}
	profile.Ombi = nil
	app.storage.SetProfileKey(profileName, profile)
	respondBool(204, true, gc)
}

type OmbiWrapper struct {
	OmbiUserByJfID func(jfID string) (map[string]interface{}, error)
	*ombiLib.Ombi
}

func (ombi *OmbiWrapper) applyProfile(user map[string]interface{}, profile map[string]interface{}) (err error) {
	for k, v := range profile {
		switch v.(type) {
		case map[string]interface{}, []interface{}:
			user[k] = v
		default:
			if v != user[k] {
				user[k] = v
			}
		}
	}
	err = ombi.ModifyUser(user)
	return
}

func (ombi *OmbiWrapper) ImportUser(jellyfinID string, req newUserDTO, profile Profile) (err error, ok bool) {
	errors, err := ombi.NewUser(req.Username, req.Password, req.Email, profile.Ombi)
	var ombiUser map[string]interface{}
	if err != nil {
		// Check if on the off chance, Ombi's user importer has already added the account.
		ombiUser, err = ombi.getImportedUser(req.Username)
		if err == nil {
			// app.info.Println(lm.Ombi + " " + lm.UserExists)
			profile.Ombi["password"] = req.Password
			err = ombi.applyProfile(ombiUser, profile.Ombi)
			if err != nil {
				err = fmt.Errorf(lm.FailedApplyProfile, lm.Ombi, req.Username, err)
			}
		} else {
			if len(errors) != 0 {
				err = fmt.Errorf("%v, %s", err, strings.Join(errors, ", "))
			}
			return
		}
	}
	ok = true
	return
}

func (ombi *OmbiWrapper) SetContactMethods(jellyfinID string, email *string, discord *DiscordUser, telegram *TelegramUser, contactPrefs *common.ContactPreferences) (err error) {
	ombiUser, err := ombi.OmbiUserByJfID(jellyfinID)
	if err != nil {
		return
	}
	if contactPrefs == nil {
		contactPrefs = &common.ContactPreferences{
			Email:    nil,
			Discord:  nil,
			Telegram: nil,
			Matrix:   nil,
		}
	}
	if emailEnabled && email != nil {
		ombiUser["emailAddress"] = *email
		err = ombi.ModifyUser(ombiUser)
		if err != nil {
			// FIXME: This is a little ugly, considering all other errors are unformatted
			err = fmt.Errorf(lm.FailedSetEmailAddress, lm.Ombi, jellyfinID, err)
			return
		}
	}

	data := make([]ombiLib.NotificationPref, 0, 2)
	if discordEnabled {
		pref := ombiLib.NotificationPref{
			Agent:  ombiLib.NotifAgentDiscord,
			UserID: ombiUser["id"].(string),
		}
		valid := false
		if contactPrefs.Discord != nil {
			pref.Enabled = *(contactPrefs.Discord)
			valid = true
		} else if discord != nil && discord.ID != "" {
			pref.Enabled = true
			valid = true
		}
		if discord != nil {
			pref.Value = discord.ID
			valid = true
		}
		if valid {
			data = append(data, pref)
		}
	}
	if telegramEnabled && telegram != nil {
		pref := ombiLib.NotificationPref{
			Agent:  ombiLib.NotifAgentTelegram,
			UserID: ombiUser["id"].(string),
		}
		if contactPrefs.Telegram != nil {
			pref.Enabled = *(contactPrefs.Telegram)
		} else if telegram != nil && telegram.Username != "" {
			pref.Enabled = true
		}
		if telegram != nil {
			pref.Value = telegram.Username
		}
		data = append(data, pref)
	}
	if len(data) > 0 {
		var resp string
		resp, err = ombi.SetNotificationPrefs(ombiUser, data)
		if err != nil {
			if resp != "" {
				err = fmt.Errorf("%v, %s", err, resp)
			}
			return
		}
	}
	return
}

func (ombi *OmbiWrapper) Name() string { return lm.Ombi }

func (ombi *OmbiWrapper) Enabled(app *appContext, profile *Profile) bool {
	return profile != nil && profile.Ombi != nil && len(profile.Ombi) != 0 && app.config.Section("ombi").Key("enabled").MustBool(false)
}
