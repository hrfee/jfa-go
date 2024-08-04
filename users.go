package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/hrfee/mediabrowser"
	"github.com/lithammer/shortuuid/v3"
)

// ReponseFunc responds to the user, generally by HTTP response
// The cases when more than this occurs are given below.
type ResponseFunc func(gc *gin.Context)

// LogFunc prints a log line once called.
type LogFunc func()

type ContactMethodConf struct {
	Email, Discord, Telegram, Matrix bool
}
type ContactMethodUsers struct {
	Email    emailStore
	Discord  DiscordUser
	Telegram TelegramVerifiedToken
	Matrix   MatrixUser
}

type ContactMethodValidation struct {
	Verified ContactMethodConf
	Users    ContactMethodUsers
}

type NewUserParams struct {
	Req                 newUserDTO
	SourceType          ActivitySource
	Source              string
	ContextForIPLogging *gin.Context
	Profile             *Profile
}

type NewUserData struct {
	Created bool
	Success bool
	User    mediabrowser.User
	Message string
	Status  int
	Log     func()
}

// Called after a new-user-creating route has done pre-steps (veryfing contact methods for example).
func (app *appContext) NewUserPostVerification(p NewUserParams) (out NewUserData) {
	// Some helper functions which will behave as our app.info/error/debug
	deferLogInfo := func(s string, args ...any) {
		out.Log = func() {
			app.info.Printf(s, args)
		}
	}
	/* deferLogDebug := func(s string, args ...any) {
		out.Log = func() {
			app.debug.Printf(s, args)
		}
	} */
	deferLogError := func(s string, args ...any) {
		out.Log = func() {
			app.err.Printf(s, args)
		}
	}

	existingUser, _, _ := app.jf.UserByName(p.Req.Username, false)
	if existingUser.Name != "" {
		out.Message = lm.UserExists
		deferLogInfo(lm.FailedCreateUser, lm.Jellyfin, p.Req.Username, out.Message)
		out.Status = 401
		return
	}

	var status int
	var err error
	out.User, status, err = app.jf.NewUser(p.Req.Username, p.Req.Password)
	if !(status == 200 || status == 204) || err != nil {
		out.Message = err.Error()
		deferLogError(lm.FailedCreateUser, lm.Jellyfin, p.Req.Username, out.Message)
		out.Status = 401
		return
	}
	out.Created = true
	// Invalidate Cache to be safe
	app.jf.CacheExpiry = time.Now()

	app.storage.SetActivityKey(shortuuid.New(), Activity{
		Type:       ActivityCreation,
		UserID:     out.User.ID,
		SourceType: p.SourceType,
		Source:     p.Source,
		InviteCode: p.Req.Code, // Left blank when an admin does this
		Value:      out.User.Name,
		Time:       time.Now(),
	}, p.ContextForIPLogging, (p.SourceType != ActivityAdmin))

	if p.Profile != nil {
		status, err = app.jf.SetPolicy(out.User.ID, p.Profile.Policy)
		if !((status == 200 || status == 204) && err == nil) {
			app.err.Printf(lm.FailedApplyTemplate, "policy", lm.Jellyfin, out.User.ID, err)
		}
		status, err = app.jf.SetConfiguration(out.User.ID, p.Profile.Configuration)
		if (status == 200 || status == 204) && err == nil {
			status, err = app.jf.SetDisplayPreferences(out.User.ID, p.Profile.Displayprefs)
		}
		if !((status == 200 || status == 204) && err == nil) {
			app.err.Printf(lm.FailedApplyTemplate, "configuration", lm.Jellyfin, out.User.ID, err)
		}

		for _, tps := range app.thirdPartyServices {
			if !tps.Enabled(app, p.Profile) {
				continue
			}
			// When ok and err != nil, its a non-fatal failure that we lot without the "FailedImportUser".
			err, ok := tps.ImportUser(out.User.ID, p.Req, *p.Profile)
			if !ok {
				app.err.Printf(lm.FailedImportUser, tps.Name(), p.Req.Username, err)
			} else if err != nil {
				app.info.Println(err)
			}
		}
	}

	// Welcome email is sent by each user of this method separately..

	out.Status = 200
	out.Success = true
	return
}

func (app *appContext) WelcomeNewUser(user mediabrowser.User, expiry time.Time) (failed bool) {
	if !app.config.Section("welcome_email").Key("enabled").MustBool(false) {
		// we didn't "fail", we "politely declined"
		// failed = true
		return
	}
	failed = true
	name := app.getAddressOrName(user.ID)
	if name == "" {
		return
	}
	msg, err := app.email.constructWelcome(user.Name, expiry, app, false)
	if err != nil {
		app.err.Printf(lm.FailedConstructWelcomeMessage, user.ID, err)
	} else if err := app.sendByID(msg, user.ID); err != nil {
		app.err.Printf(lm.FailedSendWelcomeMessage, user.ID, name, err)
	} else {
		app.info.Printf(lm.SentWelcomeMessage, user.ID, name)
		failed = false
	}
	return
}

func (app *appContext) SetUserDisabled(user mediabrowser.User, disabled bool) (err error, change bool, activityType ActivityType) {
	activityType = ActivityEnabled
	if disabled {
		activityType = ActivityDisabled
	}
	change = user.Policy.IsDisabled != disabled
	user.Policy.IsDisabled = disabled

	var status int
	status, err = app.jf.SetPolicy(user.ID, user.Policy)
	if !(status == 200 || status == 204) && err == nil {
		err = fmt.Errorf("failed (code %d)", status)
	}
	if err != nil {
		return
	}

	if app.discord != nil && app.config.Section("discord").Key("disable_enable_role").MustBool(false) {
		cmUser, ok := app.storage.GetDiscordKey(user.ID)
		if ok {
			if err := app.discord.SetRoleDisabled(cmUser.MethodID().(string), disabled); err != nil {
				app.err.Printf(lm.FailedSetDiscordMemberRole, err)
			}
		}
	}
	return
}

func (app *appContext) DeleteUser(user mediabrowser.User) (err error, deleted bool) {
	var status int
	if app.ombi != nil {
		var tpUser map[string]any
		tpUser, status, err = app.getOmbiUser(user.ID)
		if status == 200 && err == nil {
			if id, ok := tpUser["id"]; ok {
				status, err = app.ombi.DeleteUser(id.(string))
				if status != 200 && err == nil {
					err = fmt.Errorf("failed (code %d)", status)
				}
				if err != nil {
					app.err.Printf(lm.FailedDeleteUser, lm.Ombi, user.ID, err)
				}
			}
		}
	}

	if app.discord != nil && app.config.Section("discord").Key("disable_enable_role").MustBool(false) {
		cmUser, ok := app.storage.GetDiscordKey(user.ID)
		if ok {
			if err := app.discord.RemoveRole(cmUser.MethodID().(string)); err != nil {
				app.err.Printf(lm.FailedSetDiscordMemberRole, err)
			}
		}
	}

	status, err = app.jf.DeleteUser(user.ID)
	if status != 200 && status != 204 && err == nil {
		err = fmt.Errorf("failed (code %d)", status)
	}
	if err != nil {
		return
	}
	deleted = true
	return
}
