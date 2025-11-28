package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/timshannon/badgerhold/v4"
)

// @Summary Get the names of all available profile.
// @Produce json
// @Success 200 {object} getProfileNamesDTO
// @Router /profiles/names [get]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) GetProfileNames(gc *gin.Context) {
	fullProfileList := app.storage.GetProfiles()
	profiles := make([]string, len(fullProfileList))
	if len(profiles) != 0 {
		defaultProfile := app.storage.GetDefaultProfile()
		profiles[0] = defaultProfile.Name
		i := 1
		if len(fullProfileList) > 1 {
			app.storage.db.ForEach(badgerhold.Where("Name").Ne(profiles[0]), func(p *Profile) error {
				profiles[i] = p.Name
				i++
				return nil
			})
		}
	}
	resp := getProfileNamesDTO{
		Profiles: profiles,
	}
	gc.JSON(200, resp)
}

// @Summary Get all available profiles, indexed by their names.
// @Produce json
// @Success 200 {object} getProfilesDTO
// @Router /profiles [get]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) GetProfiles(gc *gin.Context) {
	out := getProfilesDTO{
		DefaultProfile: app.storage.GetDefaultProfile().Name,
		Profiles:       map[string]profileDTO{},
	}
	referralsEnabled := app.config.Section("user_page").Key("referrals").MustBool(false)
	baseInv := Invite{}
	for _, p := range app.storage.GetProfiles() {
		pdto := profileDTO{
			Admin:            p.Admin,
			LibraryAccess:    p.LibraryAccess,
			FromUser:         p.FromUser,
			Ombi:             p.Ombi != nil,
			Jellyseerr:       p.Jellyseerr.Enabled,
			ReferralsEnabled: false,
		}
		if referralsEnabled {
			err := app.storage.db.Get(p.ReferralTemplateKey, &baseInv)
			if p.ReferralTemplateKey != "" && err == nil {
				pdto.ReferralsEnabled = true
			}
		}
		out.Profiles[p.Name] = pdto
	}
	gc.JSON(200, out)
}

// @Summary Get the raw values stored in a profile (Configuration, Policy, Jellyseerr/Ombi if applicable, etc.).
// @Produce json
// @Success 200 {object} ProfileDTO
// @Failure 400 {object} boolResponse
// @Param name path string true "name of profile (url encoded if necessary)"
// @Router /profiles/raw/{name} [get]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) GetRawProfile(gc *gin.Context) {
	escapedName := gc.Param("name")
	name, err := url.QueryUnescape(escapedName)
	if err != nil {
		respondBool(400, false, gc)
		return
	}
	if profile, ok := app.storage.GetProfileKey(name); ok {
		gc.JSON(200, profile.ProfileDTO)
		return
	}
	respondBool(400, false, gc)
}

// @Summary Update the raw data of a profile (Configuration, Policy, Jellyseerr/Ombi if applicable, etc.).
// @Produce json
// @Param ProfileDTO body ProfileDTO true "Raw profile data (all of it, do not omit anything)"
// @Success 204 {object} boolResponse
// @Success 201 {object} boolResponse
// @Failure 400 {object} boolResponse
// @Router /profiles/raw/{name} [put]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) ReplaceRawProfile(gc *gin.Context) {
	escapedName := gc.Param("name")
	name, err := url.QueryUnescape(escapedName)
	if err != nil {
		respondBool(400, false, gc)
		return
	}
	existingProfile, ok := app.storage.GetProfileKey(name)
	if !ok {
		respondBool(400, false, gc)
		return
	}
	var req ProfileDTO
	gc.BindJSON(&req)
	existingProfile.ProfileDTO = req
	if req.Name == "" {
		req.Name = name
	}
	status := http.StatusNoContent
	app.storage.SetProfileKey(req.Name, existingProfile)
	if req.Name != name {
		// Name change
		app.storage.DeleteProfileKey(name)
		if discordEnabled {
			app.discord.UpdateCommands()
		}
		status = http.StatusCreated
	}
	respondBool(status, true, gc)
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
	app.info.Printf(lm.SetDefaultProfile, req.Name)
	if _, ok := app.storage.GetProfileKey(req.Name); !ok {
		msg := fmt.Sprintf(lm.FailedGetProfile, req.Name)
		app.err.Println(msg)
		respond(500, msg, gc)
		return
	}
	app.storage.db.ForEach(&badgerhold.Query{}, func(profile *Profile) error {
		if profile.Name == req.Name {
			profile.Default = true
		} else {
			profile.Default = false
		}
		app.storage.SetProfileKey(profile.Name, *profile)
		return nil
	})
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
	var req newProfileDTO
	gc.BindJSON(&req)
	app.InvalidateJellyfinCache()
	user, err := app.jf.UserByID(req.ID, false)
	if err != nil {
		app.err.Printf(lm.FailedGetUsers, lm.Jellyfin, err)
		respond(500, "Couldn't get user", gc)
		return
	}
	profile := Profile{
		FromUser:   user.Name,
		ProfileDTO: ProfileDTO{Policy: user.Policy},
		Homescreen: req.Homescreen,
	}
	app.debug.Printf(lm.CreateProfileFromUser, user.Name)
	if req.Homescreen {
		profile.Configuration = user.Configuration
		profile.Displayprefs, err = app.jf.GetDisplayPreferences(req.ID)
		if err != nil {
			app.err.Printf(lm.FailedGetJellyfinDisplayPrefs, req.ID, err)
			respond(500, "Couldn't get displayprefs", gc)
			return
		}
	}
	app.storage.SetProfileKey(req.Name, profile)
	// Refresh discord bots, profile list
	if discordEnabled {
		app.discord.UpdateCommands()
	}
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
	app.storage.DeleteProfileKey(name)
	respondBool(200, true, gc)
}

// @Summary Enable referrals for a profile, sourced from the given invite by its code.
// @Produce json
// @Param profile path string true "name of profile to enable referrals for."
// @Param invite path string true "invite code to create referral template from."
// @Param useExpiry path string true "with-expiry or none."
// @Success 200 {object} boolResponse
// @Failure 400 {object} stringResponse
// @Failure 500 {object} stringResponse
// @Router /profiles/referral/{profile}/{invite}/{useExpiry} [post]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) EnableReferralForProfile(gc *gin.Context) {
	profileName := gc.Param("profile")
	invCode := gc.Param("invite")
	useExpiry := gc.Param("useExpiry") == "with-expiry"
	inv, ok := app.storage.GetInvitesKey(invCode)
	if !ok {
		respond(400, "Invalid invite code", gc)
		app.err.Printf(lm.InvalidInviteCode, invCode)
		return
	}
	profile, ok := app.storage.GetProfileKey(profileName)
	if !ok {
		respond(400, "Invalid profile", gc)
		app.err.Printf(lm.FailedGetProfile, profileName)
		return
	}

	// Generate new code for referral template
	inv.Code = GenerateInviteCode()
	expiryDelta := inv.ValidTill.Sub(inv.Created)
	inv.Created = time.Now()
	if useExpiry {
		inv.ValidTill = inv.Created.Add(expiryDelta)
	} else {
		inv.ValidTill = inv.Created.Add(REFERRAL_EXPIRY_DAYS * 24 * time.Hour)
	}
	inv.IsReferral = true
	inv.UseReferralExpiry = useExpiry
	// Since this is a template for multiple users, ReferrerJellyfinID is not set.
	// inv.ReferrerJellyfinID = ...

	app.storage.SetInvitesKey(inv.Code, inv)

	profile.ReferralTemplateKey = inv.Code

	app.storage.SetProfileKey(profile.Name, profile)

	respondBool(200, true, gc)
}

// @Summary Disable referrals for a profile, and removes the referral template. no-op if not enabled.
// @Produce json
// @Param profile path string true "name of profile to enable referrals for."
// @Success 200 {object} boolResponse
// @Router /profiles/referral/{profile} [delete]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) DisableReferralForProfile(gc *gin.Context) {
	profileName := gc.Param("profile")
	profile, ok := app.storage.GetProfileKey(profileName)
	if !ok {
		respondBool(200, true, gc)
		return
	}

	app.storage.DeleteInvitesKey(profile.ReferralTemplateKey)

	profile.ReferralTemplateKey = ""

	app.storage.SetProfileKey(profileName, profile)

	respondBool(200, true, gc)
}
