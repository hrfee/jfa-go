package main

import (
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
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
		app.err.Printf(lm.FailedGetJellyseerrNotificationPrefs, err)
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
