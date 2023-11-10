package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/timshannon/badgerhold/v4"
)

// @Summary Get a list of profiles
// @Produce json
// @Success 200 {object} getProfilesDTO
// @Router /profiles [get]
// @Security Bearer
// @tags Profiles & Settings
func (app *appContext) GetProfiles(gc *gin.Context) {
	app.debug.Println("Profiles requested")
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
	if _, ok := app.storage.GetProfileKey(req.Name); !ok {
		app.err.Printf("Profile not found: \"%s\"", req.Name)
		respond(500, "Profile not found", gc)
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
		FromUser:   user.Name,
		Policy:     user.Policy,
		Homescreen: req.Homescreen,
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
		app.err.Printf("\"%s\": Failed to enable referrals: invite not found", profileName)
		return
	}
	profile, ok := app.storage.GetProfileKey(profileName)
	if !ok {
		respond(400, "Invalid profile", gc)
		app.err.Printf("\"%s\": Failed to enable referrals: profile not found", profileName)
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
