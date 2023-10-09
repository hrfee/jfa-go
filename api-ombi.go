package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/mediabrowser"
)

func (app *appContext) getOmbiUser(jfID string) (map[string]interface{}, int, error) {
	ombiUsers, code, err := app.ombi.GetUsers()
	if err != nil || code != 200 {
		return nil, code, err
	}
	jfUser, code, err := app.jf.UserByID(jfID, false)
	if err != nil || code != 200 {
		return nil, code, err
	}
	username := jfUser.Name
	email := ""
	if e, ok := app.storage.GetEmailsKey(jfID); ok {
		email = e.Addr
	}
	for _, ombiUser := range ombiUsers {
		ombiAddr := ""
		if a, ok := ombiUser["emailAddress"]; ok && a != nil {
			ombiAddr = a.(string)
		}
		if ombiUser["userName"].(string) == username || (ombiAddr == email && email != "") {
			return ombiUser, code, err
		}
	}
	return nil, 400, fmt.Errorf("couldn't find user")
}

// Returns a user with the given name who has been imported from Jellyfin/Emby by Ombi
func (app *appContext) getOmbiImportedUser(name string) (map[string]interface{}, int, error) {
	// Ombi User Types: 3/4 = Emby, 5 = Jellyfin
	ombiUsers, code, err := app.ombi.GetUsers()
	if err != nil || code != 200 {
		return nil, code, err
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
			return ombiUser, code, err
		}
	}
	return nil, 400, fmt.Errorf("couldn't find user")
}

// @Summary Get a list of Ombi users.
// @Produce json
// @Success 200 {object} ombiUsersDTO
// @Failure 500 {object} stringResponse
// @Router /ombi/users [get]
// @Security Bearer
// @tags Ombi
func (app *appContext) OmbiUsers(gc *gin.Context) {
	app.debug.Println("Ombi users requested")
	users, status, err := app.ombi.GetUsers()
	if err != nil || status != 200 {
		app.err.Printf("Failed to get users from Ombi (%d): %v", status, err)
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
	profileName := gc.Param("profile")
	profile, ok := app.storage.GetProfileKey(profileName)
	if !ok {
		respondBool(400, false, gc)
		return
	}
	template, code, err := app.ombi.TemplateByID(req.ID)
	if err != nil || code != 200 || len(template) == 0 {
		app.err.Printf("Couldn't get user from Ombi (%d): %v", code, err)
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
	profileName := gc.Param("profile")
	profile, ok := app.storage.GetProfileKey(profileName)
	if !ok {
		respondBool(400, false, gc)
		return
	}
	profile.Ombi = nil
	app.storage.SetProfileKey(profileName, profile)
	respondBool(204, true, gc)
}

func (app *appContext) applyOmbiProfile(user map[string]interface{}, profile map[string]interface{}) (status int, err error) {
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
	status, err = app.ombi.ModifyUser(user)
	return
}
