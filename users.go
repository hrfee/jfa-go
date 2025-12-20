package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/logger"
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
func (app *appContext) NewUserPostVerification(p NewUserParams) (out NewUserData, pendingTasks *sync.WaitGroup) {
	pendingTasks = &sync.WaitGroup{}
	// Some helper functions which will behave as our app.info/error/debug
	// And make sure we capture the correct caller location.
	deferLogInfo := func(s string, args ...any) {
		loc := logger.Lshortfile(2)
		out.Log = func() {
			app.info.PrintfNoFile(loc+" "+s, args...)
		}
	}
	/* deferLogDebug := func(s string, args ...any) {
		out.Log = func() {
			app.debug.Printf(s, args)
		}
	} */
	deferLogError := func(s string, args ...any) {
		loc := logger.Lshortfile(2)
		out.Log = func() {
			app.err.PrintfNoFile(loc+" "+s, args...)
		}
	}

	if strings.ContainsRune(p.Req.Username, '+') {
		deferLogError(lm.FailedCreateUser, lm.Jellyfin, p.Req.Username, fmt.Sprintf(lm.InvalidChar, '+'))
		out.Status = 400
		out.Message = "errorSpecialSymbols"
		return
	}

	existingUser, _ := app.jf.UserByName(p.Req.Username, false)
	if existingUser.Name != "" {
		out.Message = lm.UserExists
		deferLogInfo(lm.FailedCreateUser, lm.Jellyfin, p.Req.Username, out.Message)
		out.Status = 401
		return
	}

	var err error
	out.User, err = app.jf.NewUser(p.Req.Username, p.Req.Password)
	if err != nil {
		out.Message = err.Error()
		deferLogError(lm.FailedCreateUser, lm.Jellyfin, p.Req.Username, out.Message)
		out.Status = 401
		return
	}
	out.Created = true
	// Invalidate cache to be safe
	app.InvalidateUserCaches()

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
		err = app.jf.SetPolicy(out.User.ID, p.Profile.Policy)
		if err != nil {
			app.err.Printf(lm.FailedApplyTemplate, "policy", lm.Jellyfin, out.User.ID, err)
		}
		err = app.jf.SetConfiguration(out.User.ID, p.Profile.Configuration)
		if err == nil {
			err = app.jf.SetDisplayPreferences(out.User.ID, p.Profile.Displayprefs)
		}
		if err != nil {
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

	webhookURIs := app.config.Section("webhooks").Key("created").StringsWithShadows("|")
	if len(webhookURIs) != 0 {
		summary := app.GetUserSummary(out.User)
		for _, uri := range webhookURIs {
			pendingTasks.Add(1)
			go func() {
				app.webhooks.Send(uri, summary)
				pendingTasks.Done()
			}()
		}
	}

	// Welcome email is sent by each user of this method separately.

	out.Status = 200
	out.Success = true
	app.InvalidateWebUserCache()
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
	msg, err := app.email.constructWelcome(user.Name, expiry, false)
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

	err = app.jf.SetPolicy(user.ID, user.Policy)
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
	// FIXME: Add DeleteContactMethod to TPS
	if app.ombi != nil {
		var tpUser map[string]any
		tpUser, err = app.getOmbiUser(user.ID, nil)
		if err == nil {
			if id, ok := tpUser["id"]; ok {
				err = app.ombi.DeleteUser(id.(string))
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

	err = app.jf.DeleteUser(user.ID)
	if err != nil {
		return
	}
	deleted = true
	return
}
