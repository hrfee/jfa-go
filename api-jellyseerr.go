package main

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/common"
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
		app.err.Printf(lm.FailedGetUser, jellyseerrID, lm.Jellyseerr, err)
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
	if !profile.Jellyseerr.Enabled {
		return
	}
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

func (js *JellyseerrWrapper) SetContactMethods(jellyfinID string, email *string, discord *DiscordUser, telegram *TelegramUser, contactPrefs *common.ContactPreferences) (err error) {
	_, err = js.MustGetUser(jellyfinID)
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
	contactMethods := map[jellyseerr.NotificationsField]any{}
	if emailEnabled {
		if contactPrefs.Email != nil {
			contactMethods[jellyseerr.FieldEmailEnabled] = *(contactPrefs.Email)
		} else if email != nil && *email != "" {
			contactMethods[jellyseerr.FieldEmailEnabled] = true
		}
		if email != nil {
			err = js.ModifyMainUserSettings(jellyfinID, jellyseerr.MainUserSettings{Email: *email})
			if err != nil {
				// FIXME: This is a little ugly, considering all other errors are unformatted
				err = fmt.Errorf(lm.FailedSetEmailAddress, lm.Jellyseerr, jellyfinID, err)
				return
			}
		}
	}
	if discordEnabled {
		if contactPrefs.Discord != nil {
			contactMethods[jellyseerr.FieldDiscordEnabled] = *(contactPrefs.Discord)
		} else if discord != nil && discord.ID != "" {
			contactMethods[jellyseerr.FieldDiscordEnabled] = true
		}
		if discord != nil {
			contactMethods[jellyseerr.FieldDiscord] = discord.ID
			// Whether this is still necessary or not, i don't know.
			if discord.ID == "" {
				contactMethods[jellyseerr.FieldDiscord] = jellyseerr.BogusIdentifier
			}
		}
	}
	if telegramEnabled {
		if contactPrefs.Telegram != nil {
			contactMethods[jellyseerr.FieldTelegramEnabled] = *(contactPrefs.Telegram)
		} else if telegram != nil && telegram.ChatID != 0 {
			contactMethods[jellyseerr.FieldTelegramEnabled] = true
		}
		if telegram != nil {
			contactMethods[jellyseerr.FieldTelegram] = strconv.FormatInt(telegram.ChatID, 10)
			// Whether this is still necessary or not, i don't know.
			if telegram.ChatID == 0 {
				contactMethods[jellyseerr.FieldTelegram] = jellyseerr.BogusIdentifier
			}
		}
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
