package main

import (
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

// FIXME: First load of steps are going in NewUserFromInvite, because they're only used there.
// Make an interface{} for Require/Verify/ExistingUser which all contact daemons respect, then loop through!
/*
-- STEPS --
- Validate Invite
- Validate CAPTCHA
- Validate Password
- a) Discord  (Require, Verify, ExistingUser, ApplyRole)
  b) Telegram (Require, Verify, ExistingUser)
  c) Matrix   (Require, Verify, ExistingUser)
  d) Email    (Require, Verify, ExistingUser)
* Check for existing user
* Generate JF user
- Delete Invite
* Store Activity
* Store Email
- Store Discord/Telegram/Matrix/Label
- Notify Admin (Doesn't really matter when this happens)
* Apply Profile
* Generate JS, Ombi Users, apply profiles
* Send Welcome Email


*/

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
