package main

import (
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary Get a list of profiles
// @Produce json
// @Success 200 {object} getProfilesDTO
// @Router /profiles [get]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) GetProfiles(gc *gin.Context) {
	app.storage.loadProfiles()
	app.debug.Println("Profiles requested")
	out := getProfilesDTO{
		DefaultProfile: app.storage.defaultProfile,
		Profiles:       map[string]profileDTO{},
	}
	for name, p := range app.storage.profiles {
		out.Profiles[name] = profileDTO{
			Admin:         p.Admin,
			LibraryAccess: p.LibraryAccess,
			FromUser:      p.FromUser,
			Ombi:          p.Ombi != nil,
		}
	}
	gc.JSON(200, out)
}

// @Summary Set the default profile to use.
// @Produce json
// @Param profileChangeDTO body profileChangeDTO true "Default profile object"
// @Success 200 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /profiles/default [post]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) SetDefaultProfile(gc *gin.Context) {
	req := profileChangeDTO{}
	gc.BindJSON(&req)
	app.info.Printf("Setting default profile to \"%s\"", req.Name)
	if _, ok := app.storage.profiles[req.Name]; !ok {
		app.err.Printf("Profile not found: \"%s\"", req.Name)
		respond(500, "Profile not found", gc)
		return
	}
	for name, profile := range app.storage.profiles {
		if name == req.Name {
			profile.Admin = true
			app.storage.profiles[name] = profile
		} else {
			profile.Admin = false
		}
	}
	app.storage.defaultProfile = req.Name
	respondBool(200, true, gc)
}

// @Summary Create a profile based on a Jellyfin user's settings.
// @Produce json
// @Param newProfileDTO body newProfileDTO true "New profile object"
// @Success 200 {object} boolResponse
// @Failure 500 {object} stringResponse
// @Router /profiles [post]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) CreateProfile(gc *gin.Context) {
	app.info.Println("Profile creation requested")
	var req newProfileDTO
	gc.BindJSON(&req)
	app.jf.CacheExpiry = time.Now()
	user, status, err := app.jf.UserByID(req.ID, false)
	if !(status == 200 || status == 204) || err != nil {
		app.err.Printf("Failed to get user from Jellyfin (%d): %v", status, err)
		respond(500, "Couldn't get user", gc)
		return
	}
	profile := Profile{
		FromUser: user.Name,
		Policy:   user.Policy,
	}
	app.debug.Printf("Creating profile from user \"%s\"", user.Name)
	if req.Homescreen {
		profile.Configuration = user.Configuration
		profile.Displayprefs, status, err = app.jf.GetDisplayPreferences(req.ID)
		if !(status == 200 || status == 204) || err != nil {
			app.err.Printf("Failed to get DisplayPrefs (%d): %v", status, err)
			respond(500, "Couldn't get displayprefs", gc)
			return
		}
	}
	app.storage.loadProfiles()
	app.storage.profiles[req.Name] = profile
	app.storage.storeProfiles()
	app.storage.loadProfiles()
	respondBool(200, true, gc)
}

// @Summary Delete an existing profile
// @Produce json
// @Param profileChangeDTO body profileChangeDTO true "Delete profile object"
// @Success 200 {object} boolResponse
// @Router /profiles [delete]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) DeleteProfile(gc *gin.Context) {
	req := profileChangeDTO{}
	gc.BindJSON(&req)
	name := req.Name
	if _, ok := app.storage.profiles[name]; ok {
		if app.storage.defaultProfile == name {
			app.storage.defaultProfile = ""
		}
		delete(app.storage.profiles, name)
	}
	app.storage.storeProfiles()
	respondBool(200, true, gc)
}
