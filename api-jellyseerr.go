package main

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/jellyseerr"
	lm "github.com/hrfee/jfa-go/logmessages"
)

// @Summary Get a list of Jellyseerr users.
// @Produce json
// @Success 200 {object} ombiUsersDTO
// @Failure 500 {object} stringResponse
// @Router /jellyseerr/users [get]
// @Security Bearer
// @tags Jellyseerr
func (app *appContext) JellyseerrUsers(gc *gin.Context) {
	users, err := app.js.GetUsers()
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyseerr, err)
		respond(500, "Couldn't get users", gc)
		return
	}
	userlist := make([]ombiUser, len(users))
	i := 0
	for _, u := range users {
		userlist[i] = ombiUser{
			Name: u.Name(),
			ID:   strconv.FormatInt(u.ID, 10),
		}
		i++
	}
	gc.JSON(200, ombiUsersDTO{Users: userlist})
}

// @Summary Store Jellyseerr user template in an existing profile.
// @Produce json
// @Param id path string true "Jellyseerr ID of user to source from"
// @Param profile path string true "Name of profile to store in"
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /profiles/jellyseerr/{profile}/{id} [post]
// @Security Bearer
// @tags Jellyseerr
func (app *appContext) SetJellyseerrProfile(gc *gin.Context) {
	jellyseerrID, err := strconv.ParseInt(gc.Param("id"), 10, 64)
	if err != nil {
		respondBool(400, false, gc)
		return
	}
	escapedProfileName := gc.Param("profile")
	profileName, _ := url.QueryUnescape(escapedProfileName)
	profile, ok := app.storage.GetProfileKey(profileName)
	if !ok {
		respondBool(400, false, gc)
		return
	}
	u, err := app.js.UserByID(jellyseerrID)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyseerr, err)
		respond(500, "Couldn't get user", gc)
		return
	}
	profile.Jellyseerr.User = u.UserTemplate
	n, err := app.js.GetNotificationPreferencesByID(jellyseerrID)
	if err != nil {
		app.err.Printf(lm.FailedGetJellyseerrNotificationPrefs, gc.Param("id"), err)
		respond(500, "Couldn't get user notification prefs", gc)
		return
	}
	profile.Jellyseerr.Notifications = n.NotificationsTemplate
	profile.Jellyseerr.Enabled = true
	app.storage.SetProfileKey(profileName, profile)
	respondBool(204, true, gc)
}

// @Summary Remove jellyseerr user template from a profile.
// @Produce json
// @Param profile path string true "Name of profile to store in"
// @Success 200 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /profiles/jellyseerr/{profile} [delete]
// @Security Bearer
// @tags Jellyseerr
func (app *appContext) DeleteJellyseerrProfile(gc *gin.Context) {
	escapedProfileName := gc.Param("profile")
	profileName, _ := url.QueryUnescape(escapedProfileName)
	profile, ok := app.storage.GetProfileKey(profileName)
	if !ok {
		respondBool(400, false, gc)
		return
	}
	profile.Jellyseerr.Enabled = false
	app.storage.SetProfileKey(profileName, profile)
	respondBool(204, true, gc)
}

type JellyseerrWrapper struct {
	*jellyseerr.Jellyseerr
}

func (js *JellyseerrWrapper) ImportUser(jellyfinID string, req newUserDTO, profile Profile) (err error, ok bool) {
	// Gets existing user (not possible) or imports the given user.
	_, err = js.MustGetUser(jellyfinID)
	if err != nil {
		return
	}
	ok = true
	err = js.ApplyTemplateToUser(jellyfinID, profile.Jellyseerr.User)
	if err != nil {
		err = fmt.Errorf(lm.FailedApplyTemplate, "user", lm.Jellyseerr, jellyfinID, err)
		return
	}
	err = js.ApplyNotificationsTemplateToUser(jellyfinID, profile.Jellyseerr.Notifications)
	if err != nil {
		err = fmt.Errorf(lm.FailedApplyTemplate, "notifications", lm.Jellyseerr, jellyfinID, err)
		return
	}
	return
}

func (js *JellyseerrWrapper) AddContactMethods(jellyfinID string, req newUserDTO, discord *DiscordUser, telegram *TelegramUser) (err error) {
	_, err = js.MustGetUser(jellyfinID)
	if err != nil {
		return
	}
	contactMethods := map[jellyseerr.NotificationsField]any{}
	if emailEnabled {
		err = js.ModifyMainUserSettings(jellyfinID, jellyseerr.MainUserSettings{Email: req.Email})
		if err != nil {
			// FIXME: This is a little ugly, considering all other errors are unformatted
			err = fmt.Errorf(lm.FailedSetEmailAddress, lm.Jellyseerr, jellyfinID, err)
			return
		} else {
			contactMethods[jellyseerr.FieldEmailEnabled] = req.EmailContact
		}
	}
	if discordEnabled && discord != nil {
		contactMethods[jellyseerr.FieldDiscord] = discord.ID
		contactMethods[jellyseerr.FieldDiscordEnabled] = req.DiscordContact
	}
	if telegramEnabled && telegram != nil {
		contactMethods[jellyseerr.FieldTelegram] = telegram.ChatID
		contactMethods[jellyseerr.FieldTelegramEnabled] = req.TelegramContact
	}
	if len(contactMethods) > 0 {
		err = js.ModifyNotifications(jellyfinID, contactMethods)
		if err != nil {
			// app.err.Printf(lm.FailedSyncContactMethods, lm.Jellyseerr, err)
			return
		}
	}
	return
}

func (js *JellyseerrWrapper) Name() string { return lm.Jellyseerr }

func (js *JellyseerrWrapper) Enabled(app *appContext, profile *Profile) bool {
	return profile != nil && profile.Jellyseerr.Enabled && app.config.Section("jellyseerr").Key("enabled").MustBool(false)
}
