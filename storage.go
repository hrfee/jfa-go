package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hrfee/mediabrowser"
	"github.com/steambap/captcha"
	"github.com/timshannon/badgerhold/v4"
)

type discordStore map[string]DiscordUser
type telegramStore map[string]TelegramUser
type matrixStore map[string]MatrixUser
type emailStore map[string]EmailAddress

type UserExpiry struct {
	JellyfinID string `badgerhold:"key"`
	Expiry     time.Time
}

type Storage struct {
	timePattern string

	db_path string
	db      *badgerhold.Store

	invite_path, emails_path, policy_path, configuration_path, displayprefs_path, ombi_path, profiles_path, customEmails_path, users_path, telegram_path, discord_path, matrix_path, announcements_path, matrix_sql_path, userPage_path string
	deprecatedUserExpiries                                                                                                                                                                                                              map[string]time.Time // Map of Jellyfin User IDs to their expiry times.
	deprecatedInvites                                                                                                                                                                                                                   Invites
	profiles                                                                                                                                                                                                                            map[string]Profile
	defaultProfile                                                                                                                                                                                                                      string
	displayprefs, ombi_template                                                                                                                                                                                                         map[string]interface{}
	deprecatedEmails                                                                                                                                                                                                                    emailStore    // Map of Jellyfin User IDs to Email addresses.
	deprecatedTelegram                                                                                                                                                                                                                  telegramStore // Map of Jellyfin User IDs to telegram users.
	deprecatedDiscord                                                                                                                                                                                                                   discordStore  // Map of Jellyfin user IDs to discord users.
	deprecatedMatrix                                                                                                                                                                                                                    matrixStore   // Map of Jellyfin user IDs to Matrix users.
	customEmails                                                                                                                                                                                                                        customEmails
	userPage                                                                                                                                                                                                                            userPageContent
	policy                                                                                                                                                                                                                              mediabrowser.Policy
	configuration                                                                                                                                                                                                                       mediabrowser.Configuration
	lang                                                                                                                                                                                                                                Lang
	deprecatedAnnouncements                                                                                                                                                                                                             map[string]announcementTemplate
}

func (app *appContext) ConnectDB() {
	opts := badgerhold.DefaultOptions
	opts.Dir = app.storage.db_path
	opts.ValueDir = app.storage.db_path
	db, err := badgerhold.Open(opts)
	if err != nil {
		app.err.Fatalf("Failed to open db \"%s\": %v", app.storage.db_path, err)
	}
	app.storage.db = db
	app.info.Printf("Connected to DB \"%s\"", app.storage.db_path)
}

// GetEmails returns a copy of the store.
func (st *Storage) GetEmails() []EmailAddress {
	result := []EmailAddress{}
	err := st.db.Find(&result, &badgerhold.Query{})
	if err != nil {
		// fmt.Printf("Failed to find emails: %v\n", err)
	}
	return result
}

// GetEmailsKey returns the value stored in the store's key.
func (st *Storage) GetEmailsKey(k string) (EmailAddress, bool) {
	result := EmailAddress{}
	err := st.db.Get(k, &result)
	ok := true
	if err != nil {
		// fmt.Printf("Failed to find email: %v\n", err)
		ok = false
	}
	return result, ok
}

// SetEmailsKey stores value v in key k.
func (st *Storage) SetEmailsKey(k string, v EmailAddress) {
	v.JellyfinID = k
	err := st.db.Upsert(k, v)
	if err != nil {
		// fmt.Printf("Failed to set email: %v\n", err)
	}
}

// DeleteEmailKey deletes value at key k.
func (st *Storage) DeleteEmailsKey(k string) {
	st.db.Delete(k, EmailAddress{})
}

// GetDiscord returns a copy of the store.
func (st *Storage) GetDiscord() []DiscordUser {
	result := []DiscordUser{}
	err := st.db.Find(&result, &badgerhold.Query{})
	if err != nil {
		// fmt.Printf("Failed to find users: %v\n", err)
	}
	return result
}

// GetDiscordKey returns the value stored in the store's key.
func (st *Storage) GetDiscordKey(k string) (DiscordUser, bool) {
	result := DiscordUser{}
	err := st.db.Get(k, &result)
	ok := true
	if err != nil {
		// fmt.Printf("Failed to find user: %v\n", err)
		ok = false
	}
	return result, ok
}

// SetDiscordKey stores value v in key k.
func (st *Storage) SetDiscordKey(k string, v DiscordUser) {
	v.JellyfinID = k
	err := st.db.Upsert(k, v)
	if err != nil {
		// fmt.Printf("Failed to set user: %v\n", err)
	}
}

// DeleteDiscordKey deletes value at key k.
func (st *Storage) DeleteDiscordKey(k string) {
	st.db.Delete(k, DiscordUser{})
}

// GetTelegram returns a copy of the store.
func (st *Storage) GetTelegram() []TelegramUser {
	result := []TelegramUser{}
	err := st.db.Find(&result, &badgerhold.Query{})
	if err != nil {
		// fmt.Printf("Failed to find users: %v\n", err)
	}
	return result
}

// GetTelegramKey returns the value stored in the store's key.
func (st *Storage) GetTelegramKey(k string) (TelegramUser, bool) {
	result := TelegramUser{}
	err := st.db.Get(k, &result)
	ok := true
	if err != nil {
		// fmt.Printf("Failed to find user: %v\n", err)
		ok = false
	}
	return result, ok
}

// SetTelegramKey stores value v in key k.
func (st *Storage) SetTelegramKey(k string, v TelegramUser) {
	v.JellyfinID = k
	err := st.db.Upsert(k, v)
	if err != nil {
		// fmt.Printf("Failed to set user: %v\n", err)
	}
}

// DeleteTelegramKey deletes value at key k.
func (st *Storage) DeleteTelegramKey(k string) {
	st.db.Delete(k, TelegramUser{})
}

// GetMatrix returns a copy of the store.
func (st *Storage) GetMatrix() []MatrixUser {
	result := []MatrixUser{}
	err := st.db.Find(&result, &badgerhold.Query{})
	if err != nil {
		// fmt.Printf("Failed to find users: %v\n", err)
	}
	return result
}

// GetMatrixKey returns the value stored in the store's key.
func (st *Storage) GetMatrixKey(k string) (MatrixUser, bool) {
	result := MatrixUser{}
	err := st.db.Get(k, &result)
	ok := true
	if err != nil {
		// fmt.Printf("Failed to find user: %v\n", err)
		ok = false
	}
	return result, ok
}

// SetMatrixKey stores value v in key k.
func (st *Storage) SetMatrixKey(k string, v MatrixUser) {
	v.JellyfinID = k
	err := st.db.Upsert(k, v)
	if err != nil {
		// fmt.Printf("Failed to set user: %v\n", err)
	}
}

// DeleteMatrixKey deletes value at key k.
func (st *Storage) DeleteMatrixKey(k string) {
	st.db.Delete(k, MatrixUser{})
}

// GetInvites returns a copy of the store.
func (st *Storage) GetInvites() []Invite {
	result := []Invite{}
	err := st.db.Find(&result, &badgerhold.Query{})
	if err != nil {
		// fmt.Printf("Failed to find invites: %v\n", err)
	}
	return result
}

// GetInvitesKey returns the value stored in the store's key.
func (st *Storage) GetInvitesKey(k string) (Invite, bool) {
	result := Invite{}
	err := st.db.Get(k, &result)
	ok := true
	if err != nil {
		// fmt.Printf("Failed to find invite: %v\n", err)
		ok = false
	}
	return result, ok
}

// SetInvitesKey stores value v in key k.
func (st *Storage) SetInvitesKey(k string, v Invite) {
	v.Code = k
	err := st.db.Upsert(k, v)
	if err != nil {
		// fmt.Printf("Failed to set invite: %v\n", err)
	}
}

// DeleteInvitesKey deletes value at key k.
func (st *Storage) DeleteInvitesKey(k string) {
	st.db.Delete(k, Invite{})
}

// GetAnnouncements returns a copy of the store.
func (st *Storage) GetAnnouncements() []announcementTemplate {
	result := []announcementTemplate{}
	err := st.db.Find(&result, &badgerhold.Query{})
	if err != nil {
		// fmt.Printf("Failed to find announcements: %v\n", err)
	}
	return result
}

// GetAnnouncementsKey returns the value stored in the store's key.
func (st *Storage) GetAnnouncementsKey(k string) (announcementTemplate, bool) {
	result := announcementTemplate{}
	err := st.db.Get(k, &result)
	ok := true
	if err != nil {
		// fmt.Printf("Failed to find announcement: %v\n", err)
		ok = false
	}
	return result, ok
}

// SetAnnouncementsKey stores value v in key k.
func (st *Storage) SetAnnouncementsKey(k string, v announcementTemplate) {
	err := st.db.Upsert(k, v)
	if err != nil {
		// fmt.Printf("Failed to set announcement: %v\n", err)
	}
}

// DeleteAnnouncementsKey deletes value at key k.
func (st *Storage) DeleteAnnouncementsKey(k string) {
	st.db.Delete(k, announcementTemplate{})
}

// GetUserExpiries returns a copy of the store.
func (st *Storage) GetUserExpiries() []UserExpiry {
	result := []UserExpiry{}
	err := st.db.Find(&result, &badgerhold.Query{})
	if err != nil {
		// fmt.Printf("Failed to find expiries: %v\n", err)
	}
	return result
}

// GetUserExpiryKey returns the value stored in the store's key.
func (st *Storage) GetUserExpiryKey(k string) (UserExpiry, bool) {
	result := UserExpiry{}
	err := st.db.Get(k, &result)
	ok := true
	if err != nil {
		// fmt.Printf("Failed to find expiry: %v\n", err)
		ok = false
	}
	return result, ok
}

// SetUserExpiryKey stores value v in key k.
func (st *Storage) SetUserExpiryKey(k string, v UserExpiry) {
	v.JellyfinID = k
	err := st.db.Upsert(k, v)
	if err != nil {
		// fmt.Printf("Failed to set expiry: %v\n", err)
	}
}

// DeleteUserExpiryKey deletes value at key k.
func (st *Storage) DeleteUserExpiryKey(k string) {
	st.db.Delete(k, UserExpiry{})
}

type TelegramUser struct {
	JellyfinID string `badgerhold:"key"`
	ChatID     int64  `badgerhold:"index"`
	Username   string `badgerhold:"index"`
	Lang       string
	Contact    bool // Whether to contact through telegram or not
}

type DiscordUser struct {
	ChannelID     string
	ID            string `badgerhold:"index"`
	Username      string `badgerhold:"index"`
	Discriminator string
	Lang          string
	Contact       bool
	JellyfinID    string `json:"-" badgerhold:"key"` // Used internally in discord.go
}

type EmailAddress struct {
	Addr       string `badgerhold:"index"`
	Label      string // User Label.
	Contact    bool
	Admin      bool   // Whether or not user is jfa-go admin.
	JellyfinID string `badgerhold:"key"`
}

type customEmails struct {
	UserCreated       customContent `json:"userCreated"`
	InviteExpiry      customContent `json:"inviteExpiry"`
	PasswordReset     customContent `json:"passwordReset"`
	UserDeleted       customContent `json:"userDeleted"`
	UserDisabled      customContent `json:"userDisabled"`
	UserEnabled       customContent `json:"userEnabled"`
	InviteEmail       customContent `json:"inviteEmail"`
	WelcomeEmail      customContent `json:"welcomeEmail"`
	EmailConfirmation customContent `json:"emailConfirmation"`
	UserExpired       customContent `json:"userExpired"`
}

type customContent struct {
	Enabled      bool     `json:"enabled,omitempty"`
	Content      string   `json:"content"`
	Variables    []string `json:"variables,omitempty"`
	Conditionals []string `json:"conditionals,omitempty"`
}

type userPageContent struct {
	Login customContent `json:"login"`
	Page  customContent `json:"page"`
}

// timePattern: %Y-%m-%dT%H:%M:%S.%f

type Profile struct {
	Admin         bool                       `json:"admin,omitempty"`
	LibraryAccess string                     `json:"libraries,omitempty"`
	FromUser      string                     `json:"fromUser,omitempty"`
	Policy        mediabrowser.Policy        `json:"policy,omitempty"`
	Configuration mediabrowser.Configuration `json:"configuration,omitempty"`
	Displayprefs  map[string]interface{}     `json:"displayprefs,omitempty"`
	Default       bool                       `json:"default,omitempty"`
	Ombi          map[string]interface{}     `json:"ombi,omitempty"`
}

type Invite struct {
	Code          string    `badgerhold:"key"`
	Created       time.Time `json:"created"`
	NoLimit       bool      `json:"no-limit"`
	RemainingUses int       `json:"remaining-uses"`
	ValidTill     time.Time `json:"valid_till"`
	UserExpiry    bool      `json:"user-duration"`
	UserMonths    int       `json:"user-months,omitempty"`
	UserDays      int       `json:"user-days,omitempty"`
	UserHours     int       `json:"user-hours,omitempty"`
	UserMinutes   int       `json:"user-minutes,omitempty"`
	SendTo        string    `json:"email"`
	// Used to be stored as formatted time, now as Unix.
	UsedBy   [][]string                 `json:"used-by"`
	Notify   map[string]map[string]bool `json:"notify"`
	Profile  string                     `json:"profile"`
	Label    string                     `json:"label,omitempty"`
	Captchas map[string]*captcha.Data   // Map of Captcha IDs to answers
}

type Lang struct {
	AdminPath         string
	chosenAdminLang   string
	Admin             adminLangs
	AdminJSON         map[string]string
	UserPath          string
	chosenUserLang    string
	User              userLangs
	PasswordResetPath string
	chosenPWRLang     string
	PasswordReset     pwrLangs
	EmailPath         string
	chosenEmailLang   string
	Email             emailLangs
	CommonPath        string
	Common            commonLangs
	SetupPath         string
	Setup             setupLangs
	// Telegram translations are also used for Discord bots (and likely future ones).
	chosenTelegramLang string
	TelegramPath       string
	Telegram           telegramLangs
}

func (st *Storage) loadLang(filesystems ...fs.FS) (err error) {
	err = st.loadLangCommon(filesystems...)
	if err != nil {
		return
	}
	err = st.loadLangAdmin(filesystems...)
	if err != nil {
		return
	}
	err = st.loadLangEmail(filesystems...)
	if err != nil {
		return
	}
	err = st.loadLangUser(filesystems...)
	if err != nil {
		return
	}
	err = st.loadLangPWR(filesystems...)
	if err != nil {
		return
	}
	err = st.loadLangTelegram(filesystems...)
	return
}

// The following patch* functions fill in a language with missing values
// from a list of other sources in a preferred order.
// languages to patch from should be in decreasing priority,
// E.g: If to = fr-be, from = [fr-fr, en-us].
func (common *commonLangs) patchCommonStrings(to *langSection, from ...string) {
	if *to == nil {
		*to = langSection{}
	}
	for n, ev := range (*common)[from[len(from)-1]].Strings {
		if v, ok := (*to)[n]; !ok || v == "" {
			i := 0
			for i < len(from)-1 {
				ev, ok = (*common)[from[i]].Strings[n]
				if ok && ev != "" {
					break
				}
				i++
			}
			(*to)[n] = ev
		}
	}
}

func (common *commonLangs) patchCommonNotifications(to *langSection, from ...string) {
	if *to == nil {
		*to = langSection{}
	}
	for n, ev := range (*common)[from[len(from)-1]].Notifications {
		if v, ok := (*to)[n]; !ok || v == "" {
			i := 0
			for i < len(from)-1 {
				ev, ok = (*common)[from[i]].Notifications[n]
				if ok && ev != "" {
					break
				}
				i++
			}
			(*to)[n] = ev
		}
	}
}

func (common *commonLangs) patchCommonQuantityStrings(to *map[string]quantityString, from ...string) {
	if *to == nil {
		*to = map[string]quantityString{}
	}
	for n, ev := range (*common)[from[len(from)-1]].QuantityStrings {
		if v, ok := (*to)[n]; !ok || (v.Singular == "" && v.Plural == "") {
			i := 0
			for i < len(from)-1 {
				ev, ok = (*common)[from[i]].QuantityStrings[n]
				if ok && ev.Singular != "" && ev.Plural != "" {
					break
				}
				i++
			}
			(*to)[n] = ev
		}
	}
}

func patchLang(to *langSection, from ...*langSection) {
	if *to == nil {
		*to = langSection{}
	}
	for n, ev := range *from[len(from)-1] {
		if v, ok := (*to)[n]; !ok || v == "" {
			i := 0
			for i < len(from)-1 {
				ev, ok = (*from[i])[n]
				if ok && ev != "" {
					break
				}
				i++
			}
			(*to)[n] = ev
		}
	}
}

func patchQuantityStrings(to *map[string]quantityString, from ...*map[string]quantityString) {
	if *to == nil {
		*to = map[string]quantityString{}
	}
	for n, ev := range *from[len(from)-1] {
		qs, ok := (*to)[n]
		if !ok || qs.Singular == "" || qs.Plural == "" {
			i := 0
			subOk := false
			for i < len(from)-1 {
				ev, subOk = (*from[i])[n]
				if subOk && ev.Singular != "" && ev.Plural != "" {
					break
				}
				i++
			}
			if !ok {
				(*to)[n] = ev
				continue
			} else if qs.Singular == "" {
				qs.Singular = ev.Singular
			} else if qs.Plural == "" {
				qs.Plural = ev.Plural
			}
			(*to)[n] = qs
		}
	}
}

type loadLangFunc func(fsIndex int, name string) error

func (st *Storage) loadLangCommon(filesystems ...fs.FS) error {
	st.lang.Common = map[string]commonLang{}
	var english commonLang
	loadedLangs := make([]map[string]bool, len(filesystems))
	var load loadLangFunc
	load = func(fsIndex int, fname string) error {
		filesystem := filesystems[fsIndex]
		index := strings.TrimSuffix(fname, filepath.Ext(fname))
		lang := commonLang{}
		f, err := fs.ReadFile(filesystem, FSJoin(st.lang.CommonPath, fname))
		if err != nil {
			return err
		}
		if substituteStrings != "" {
			f = []byte(strings.ReplaceAll(string(f), "Jellyfin", substituteStrings))
		}
		err = json.Unmarshal(f, &lang)
		if err != nil {
			return err
		}
		if fname != "en-us.json" {
			if lang.Meta.Fallback != "" {
				fallback, ok := st.lang.Common[lang.Meta.Fallback]
				err = nil
				if !ok {
					err = load(fsIndex, lang.Meta.Fallback+".json")
					fallback = st.lang.Common[lang.Meta.Fallback]
				}
				if err == nil {
					loadedLangs[fsIndex][lang.Meta.Fallback+".json"] = true
					patchLang(&lang.Strings, &fallback.Strings, &english.Strings)
					patchLang(&lang.Notifications, &fallback.Notifications, &english.Notifications)
					patchQuantityStrings(&lang.QuantityStrings, &fallback.QuantityStrings, &english.QuantityStrings)
				}
			}
			if (lang.Meta.Fallback != "" && err != nil) || lang.Meta.Fallback == "" {
				patchLang(&lang.Strings, &english.Strings)
				patchLang(&lang.Notifications, &english.Notifications)
				patchQuantityStrings(&lang.QuantityStrings, &english.QuantityStrings)
			}
		}
		st.lang.Common[index] = lang
		return nil
	}
	engFound := false
	var err error
	for i := range filesystems {
		loadedLangs[i] = map[string]bool{}
		err = load(i, "en-us.json")
		if err == nil {
			engFound = true
		}
		loadedLangs[i]["en-us.json"] = true
	}
	if !engFound {
		return err
	}
	english = st.lang.Common["en-us"]
	commonLoaded := false
	for i := range filesystems {
		files, err := fs.ReadDir(filesystems[i], st.lang.CommonPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !loadedLangs[i][f.Name()] {
				err = load(i, f.Name())
				if err == nil {
					commonLoaded = true
					loadedLangs[i][f.Name()] = true
				}
			}
		}
	}
	if !commonLoaded {
		return err
	}
	return nil
}

func (st *Storage) loadLangAdmin(filesystems ...fs.FS) error {
	st.lang.Admin = map[string]adminLang{}
	var english adminLang
	loadedLangs := make([]map[string]bool, len(filesystems))
	var load loadLangFunc
	load = func(fsIndex int, fname string) error {
		filesystem := filesystems[fsIndex]
		index := strings.TrimSuffix(fname, filepath.Ext(fname))
		lang := adminLang{}
		f, err := fs.ReadFile(filesystem, FSJoin(st.lang.AdminPath, fname))
		if err != nil {
			return err
		}
		if substituteStrings != "" {
			f = []byte(strings.ReplaceAll(string(f), "Jellyfin", substituteStrings))
		}
		err = json.Unmarshal(f, &lang)
		if err != nil {
			return err
		}
		st.lang.Common.patchCommonStrings(&lang.Strings, index)
		st.lang.Common.patchCommonNotifications(&lang.Notifications, index)
		if fname != "en-us.json" {
			if lang.Meta.Fallback != "" {
				fallback, ok := st.lang.Admin[lang.Meta.Fallback]
				err = nil
				if !ok {
					err = load(fsIndex, lang.Meta.Fallback+".json")
					fallback = st.lang.Admin[lang.Meta.Fallback]
				}
				if err == nil {
					loadedLangs[fsIndex][lang.Meta.Fallback+".json"] = true
					patchLang(&lang.Strings, &fallback.Strings, &english.Strings)
					patchLang(&lang.Notifications, &fallback.Notifications, &english.Notifications)
					patchQuantityStrings(&lang.QuantityStrings, &fallback.QuantityStrings, &english.QuantityStrings)
				}
			}
			if (lang.Meta.Fallback != "" && err != nil) || lang.Meta.Fallback == "" {
				patchLang(&lang.Strings, &english.Strings)
				patchLang(&lang.Notifications, &english.Notifications)
				patchQuantityStrings(&lang.QuantityStrings, &english.QuantityStrings)
			}
		}
		stringAdmin, err := json.Marshal(lang)
		if err != nil {
			return err
		}
		lang.JSON = string(stringAdmin)
		st.lang.Admin[index] = lang
		return nil
	}
	engFound := false
	var err error
	for i := range filesystems {
		loadedLangs[i] = map[string]bool{}
		err = load(i, "en-us.json")
		if err == nil {
			engFound = true
		}
		loadedLangs[i]["en-us.json"] = true
	}
	if !engFound {
		return err
	}
	english = st.lang.Admin["en-us"]
	adminLoaded := false
	for i := range filesystems {
		files, err := fs.ReadDir(filesystems[i], st.lang.AdminPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !loadedLangs[i][f.Name()] {
				err = load(i, f.Name())
				if err == nil {
					adminLoaded = true
					loadedLangs[i][f.Name()] = true
				}
			}
		}
	}
	if !adminLoaded {
		return err
	}
	return nil
}

func (st *Storage) loadLangUser(filesystems ...fs.FS) error {
	st.lang.User = map[string]userLang{}
	var english userLang
	loadedLangs := make([]map[string]bool, len(filesystems))
	var load loadLangFunc
	load = func(fsIndex int, fname string) error {
		filesystem := filesystems[fsIndex]
		index := strings.TrimSuffix(fname, filepath.Ext(fname))
		lang := userLang{}
		f, err := fs.ReadFile(filesystem, FSJoin(st.lang.UserPath, fname))
		if err != nil {
			return err
		}
		if substituteStrings != "" {
			f = []byte(strings.ReplaceAll(string(f), "Jellyfin", substituteStrings))
		}
		err = json.Unmarshal(f, &lang)
		if err != nil {
			return err
		}
		st.lang.Common.patchCommonStrings(&lang.Strings, index)
		st.lang.Common.patchCommonNotifications(&lang.Notifications, index)
		st.lang.Common.patchCommonQuantityStrings(&lang.QuantityStrings, index)
		// turns out, a lot of email strings are useful on the user page.
		emailLang := []langSection{st.lang.Email[index].WelcomeEmail, st.lang.Email[index].UserDisabled, st.lang.Email[index].UserExpired}
		for _, v := range emailLang {
			patchLang(&lang.Strings, &v)
		}
		if fname != "en-us.json" {
			if lang.Meta.Fallback != "" {
				fallback, ok := st.lang.User[lang.Meta.Fallback]
				err = nil
				if !ok {
					err = load(fsIndex, lang.Meta.Fallback+".json")
					fallback = st.lang.User[lang.Meta.Fallback]
				}
				if err == nil {
					loadedLangs[fsIndex][lang.Meta.Fallback+".json"] = true
					patchLang(&lang.Strings, &fallback.Strings, &english.Strings)
					patchLang(&lang.Notifications, &fallback.Notifications, &english.Notifications)
					patchQuantityStrings(&lang.ValidationStrings, &fallback.ValidationStrings, &english.ValidationStrings)
				}
			}
			if (lang.Meta.Fallback != "" && err != nil) || lang.Meta.Fallback == "" {
				patchLang(&lang.Strings, &english.Strings)
				patchLang(&lang.Notifications, &english.Notifications)
				patchQuantityStrings(&lang.ValidationStrings, &english.ValidationStrings)
			}
		}
		notifications, err := json.Marshal(lang.Notifications)
		if err != nil {
			return err
		}
		validationStrings, err := json.Marshal(lang.ValidationStrings)
		if err != nil {
			return err
		}
		userJSON, err := json.Marshal(lang)
		if err != nil {
			return err
		}
		lang.notificationsJSON = string(notifications)
		lang.validationStringsJSON = string(validationStrings)
		lang.JSON = string(userJSON)
		st.lang.User[index] = lang
		return nil
	}
	engFound := false
	var err error
	for i := range filesystems {
		loadedLangs[i] = map[string]bool{}
		err = load(i, "en-us.json")
		if err == nil {
			engFound = true
		}
		loadedLangs[i]["en-us.json"] = true
	}
	if !engFound {
		return err
	}
	english = st.lang.User["en-us"]
	userLoaded := false
	for i := range filesystems {
		files, err := fs.ReadDir(filesystems[i], st.lang.UserPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !loadedLangs[i][f.Name()] {
				err = load(i, f.Name())
				if err == nil {
					userLoaded = true
					loadedLangs[i][f.Name()] = true
				}
			}
		}
	}
	if !userLoaded {
		return err
	}
	return nil
}

func (st *Storage) loadLangPWR(filesystems ...fs.FS) error {
	st.lang.PasswordReset = map[string]pwrLang{}
	var english pwrLang
	loadedLangs := make([]map[string]bool, len(filesystems))
	var load loadLangFunc
	load = func(fsIndex int, fname string) error {
		filesystem := filesystems[fsIndex]
		index := strings.TrimSuffix(fname, filepath.Ext(fname))
		lang := pwrLang{}
		f, err := fs.ReadFile(filesystem, FSJoin(st.lang.PasswordResetPath, fname))
		if err != nil {
			return err
		}
		if substituteStrings != "" {
			f = []byte(strings.ReplaceAll(string(f), "Jellyfin", substituteStrings))
		}
		err = json.Unmarshal(f, &lang)
		if err != nil {
			return err
		}
		st.lang.Common.patchCommonStrings(&lang.Strings, index)
		if fname != "en-us.json" {
			if lang.Meta.Fallback != "" {
				fallback, ok := st.lang.PasswordReset[lang.Meta.Fallback]
				err = nil
				if !ok {
					err = load(fsIndex, lang.Meta.Fallback+".json")
					fallback = st.lang.PasswordReset[lang.Meta.Fallback]
				}
				if err == nil {
					patchLang(&lang.Strings, &fallback.Strings, &english.Strings)
				}
			}
			if (lang.Meta.Fallback != "" && err != nil) || lang.Meta.Fallback == "" {
				patchLang(&lang.Strings, &english.Strings)
			}
		}
		st.lang.PasswordReset[index] = lang
		return nil
	}
	engFound := false
	var err error
	for i := range filesystems {
		loadedLangs[i] = map[string]bool{}
		err = load(i, "en-us.json")
		if err == nil {
			engFound = true
		}
		loadedLangs[i]["en-us.json"] = true
	}
	if !engFound {
		return err
	}
	english = st.lang.PasswordReset["en-us"]
	userLoaded := false
	for i := range filesystems {
		files, err := fs.ReadDir(filesystems[i], st.lang.PasswordResetPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !loadedLangs[i][f.Name()] {
				err = load(i, f.Name())
				if err == nil {
					userLoaded = true
					loadedLangs[i][f.Name()] = true
				}
			}
		}
	}
	if !userLoaded {
		return err
	}
	return nil
}

func (st *Storage) loadLangEmail(filesystems ...fs.FS) error {
	st.lang.Email = map[string]emailLang{}
	var english emailLang
	loadedLangs := make([]map[string]bool, len(filesystems))
	var load loadLangFunc
	load = func(fsIndex int, fname string) error {
		filesystem := filesystems[fsIndex]
		index := strings.TrimSuffix(fname, filepath.Ext(fname))
		lang := emailLang{}
		f, err := fs.ReadFile(filesystem, FSJoin(st.lang.EmailPath, fname))
		if err != nil {
			return err
		}
		if substituteStrings != "" {
			f = []byte(strings.ReplaceAll(string(f), "Jellyfin", substituteStrings))
		}
		err = json.Unmarshal(f, &lang)
		if err != nil {
			return err
		}
		st.lang.Common.patchCommonStrings(&lang.Strings, index)
		if fname != "en-us.json" {
			if lang.Meta.Fallback != "" {
				fallback, ok := st.lang.Email[lang.Meta.Fallback]
				err = nil
				if !ok {
					err = load(fsIndex, lang.Meta.Fallback+".json")
					fallback = st.lang.Email[lang.Meta.Fallback]
				}
				if err == nil {
					loadedLangs[fsIndex][lang.Meta.Fallback+".json"] = true
					patchLang(&lang.UserCreated, &fallback.UserCreated, &english.UserCreated)
					patchLang(&lang.InviteExpiry, &fallback.InviteExpiry, &english.InviteExpiry)
					patchLang(&lang.PasswordReset, &fallback.PasswordReset, &english.PasswordReset)
					patchLang(&lang.UserDeleted, &fallback.UserDeleted, &english.UserDeleted)
					patchLang(&lang.UserDisabled, &fallback.UserDisabled, &english.UserDisabled)
					patchLang(&lang.UserEnabled, &fallback.UserEnabled, &english.UserEnabled)
					patchLang(&lang.InviteEmail, &fallback.InviteEmail, &english.InviteEmail)
					patchLang(&lang.WelcomeEmail, &fallback.WelcomeEmail, &english.WelcomeEmail)
					patchLang(&lang.EmailConfirmation, &fallback.EmailConfirmation, &english.EmailConfirmation)
					patchLang(&lang.UserExpired, &fallback.UserExpired, &english.UserExpired)
					patchLang(&lang.Strings, &fallback.Strings, &english.Strings)
				}
			}
			if (lang.Meta.Fallback != "" && err != nil) || lang.Meta.Fallback == "" {
				patchLang(&lang.UserCreated, &english.UserCreated)
				patchLang(&lang.InviteExpiry, &english.InviteExpiry)
				patchLang(&lang.PasswordReset, &english.PasswordReset)
				patchLang(&lang.UserDeleted, &english.UserDeleted)
				patchLang(&lang.UserDisabled, &english.UserDisabled)
				patchLang(&lang.UserEnabled, &english.UserEnabled)
				patchLang(&lang.InviteEmail, &english.InviteEmail)
				patchLang(&lang.WelcomeEmail, &english.WelcomeEmail)
				patchLang(&lang.EmailConfirmation, &english.EmailConfirmation)
				patchLang(&lang.UserExpired, &english.UserExpired)
				patchLang(&lang.Strings, &english.Strings)
			}
		}
		st.lang.Email[index] = lang
		return nil
	}
	engFound := false
	var err error
	for i := range filesystems {
		loadedLangs[i] = map[string]bool{}
		err = load(i, "en-us.json")
		if err == nil {
			engFound = true
		}
		loadedLangs[i]["en-us.json"] = true
	}
	if !engFound {
		return err
	}
	english = st.lang.Email["en-us"]
	emailLoaded := false
	for i := range filesystems {
		files, err := fs.ReadDir(filesystems[i], st.lang.EmailPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !loadedLangs[i][f.Name()] {
				err = load(i, f.Name())
				if err == nil {
					emailLoaded = true
					loadedLangs[i][f.Name()] = true
				}
			}
		}
	}
	if !emailLoaded {
		return err
	}
	return nil
}

func (st *Storage) loadLangTelegram(filesystems ...fs.FS) error {
	st.lang.Telegram = map[string]telegramLang{}
	var english telegramLang
	loadedLangs := make([]map[string]bool, len(filesystems))
	var load loadLangFunc
	load = func(fsIndex int, fname string) error {
		filesystem := filesystems[fsIndex]
		index := strings.TrimSuffix(fname, filepath.Ext(fname))
		lang := telegramLang{}
		f, err := fs.ReadFile(filesystem, FSJoin(st.lang.TelegramPath, fname))
		if err != nil {
			return err
		}
		if substituteStrings != "" {
			f = []byte(strings.ReplaceAll(string(f), "Jellyfin", substituteStrings))
		}
		err = json.Unmarshal(f, &lang)
		if err != nil {
			return err
		}
		st.lang.Common.patchCommonStrings(&lang.Strings, index)
		if fname != "en-us.json" {
			if lang.Meta.Fallback != "" {
				fallback, ok := st.lang.Telegram[lang.Meta.Fallback]
				err = nil
				if !ok {
					err = load(fsIndex, lang.Meta.Fallback+".json")
					fallback = st.lang.Telegram[lang.Meta.Fallback]
				}
				if err == nil {
					loadedLangs[fsIndex][lang.Meta.Fallback+".json"] = true
					patchLang(&lang.Strings, &fallback.Strings, &english.Strings)
				}
			}
			if (lang.Meta.Fallback != "" && err != nil) || lang.Meta.Fallback == "" {
				patchLang(&lang.Strings, &english.Strings)
			}
		}
		st.lang.Telegram[index] = lang
		return nil
	}
	engFound := false
	var err error
	for i := range filesystems {
		loadedLangs[i] = map[string]bool{}
		err = load(i, "en-us.json")
		if err == nil {
			engFound = true
		}
		loadedLangs[i]["en-us.json"] = true
	}
	if !engFound {
		return err
	}
	english = st.lang.Telegram["en-us"]
	telegramLoaded := false
	for i := range filesystems {
		files, err := fs.ReadDir(filesystems[i], st.lang.TelegramPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !loadedLangs[i][f.Name()] {
				err = load(i, f.Name())
				if err == nil {
					telegramLoaded = true
					loadedLangs[i][f.Name()] = true
				}
			}
		}
	}
	if !telegramLoaded {
		return err
	}
	return nil
}

type Invites map[string]Invite

func (st *Storage) loadInvites() error {
	return loadJSON(st.invite_path, &st.deprecatedInvites)
}

func (st *Storage) storeInvites() error {
	return storeJSON(st.invite_path, st.deprecatedInvites)
}

func (st *Storage) loadUserExpiries() error {
	if st.deprecatedUserExpiries == nil {
		st.deprecatedUserExpiries = map[string]time.Time{}
	}
	temp := map[string]time.Time{}
	err := loadJSON(st.users_path, &temp)
	if err != nil {
		return err
	}
	for id, t1 := range temp {
		if _, ok := st.deprecatedUserExpiries[id]; !ok {
			st.deprecatedUserExpiries[id] = t1
		}
	}
	return nil
}

func (st *Storage) storeUserExpiries() error {
	return storeJSON(st.users_path, st.deprecatedUserExpiries)
}

func (st *Storage) loadEmails() error {
	return loadJSON(st.emails_path, &st.deprecatedEmails)
}

func (st *Storage) storeEmails() error {
	return storeJSON(st.emails_path, st.deprecatedEmails)
}

func (st *Storage) loadTelegramUsers() error {
	return loadJSON(st.telegram_path, &st.deprecatedTelegram)
}

func (st *Storage) storeTelegramUsers() error {
	return storeJSON(st.telegram_path, st.deprecatedTelegram)
}

func (st *Storage) loadDiscordUsers() error {
	return loadJSON(st.discord_path, &st.deprecatedDiscord)
}

func (st *Storage) storeDiscordUsers() error {
	return storeJSON(st.discord_path, st.deprecatedDiscord)
}

func (st *Storage) loadMatrixUsers() error {
	return loadJSON(st.matrix_path, &st.deprecatedMatrix)
}

func (st *Storage) storeMatrixUsers() error {
	return storeJSON(st.matrix_path, st.deprecatedMatrix)
}

func (st *Storage) loadCustomEmails() error {
	return loadJSON(st.customEmails_path, &st.customEmails)
}

func (st *Storage) storeCustomEmails() error {
	return storeJSON(st.customEmails_path, st.customEmails)
}

func (st *Storage) loadUserPageContent() error {
	return loadJSON(st.userPage_path, &st.userPage)
}

func (st *Storage) storeUserPageContent() error {
	return storeJSON(st.userPage_path, st.userPage)
}

func (st *Storage) loadPolicy() error {
	return loadJSON(st.policy_path, &st.policy)
}

func (st *Storage) storePolicy() error {
	return storeJSON(st.policy_path, st.policy)
}

func (st *Storage) loadConfiguration() error {
	return loadJSON(st.configuration_path, &st.configuration)
}

func (st *Storage) storeConfiguration() error {
	return storeJSON(st.configuration_path, st.configuration)
}

func (st *Storage) loadDisplayprefs() error {
	return loadJSON(st.displayprefs_path, &st.displayprefs)
}

func (st *Storage) storeDisplayprefs() error {
	return storeJSON(st.displayprefs_path, st.displayprefs)
}

func (st *Storage) loadOmbiTemplate() error {
	return loadJSON(st.ombi_path, &st.ombi_template)
}

func (st *Storage) storeOmbiTemplate() error {
	return storeJSON(st.ombi_path, st.ombi_template)
}

func (st *Storage) loadAnnouncements() error {
	return loadJSON(st.announcements_path, &st.deprecatedAnnouncements)
}

func (st *Storage) storeAnnouncements() error {
	return storeJSON(st.announcements_path, st.deprecatedAnnouncements)
}

func (st *Storage) loadProfiles() error {
	err := loadJSON(st.profiles_path, &st.profiles)
	for name, profile := range st.profiles {
		if profile.Default {
			st.defaultProfile = name
		}
		change := false
		if profile.Policy.IsAdministrator != profile.Admin {
			change = true
		}
		profile.Admin = profile.Policy.IsAdministrator
		if profile.Policy.EnabledFolders != nil {
			length := len(profile.Policy.EnabledFolders)
			if length == 0 {
				profile.LibraryAccess = "All"
			} else {
				profile.LibraryAccess = strconv.Itoa(length)
			}
			change = true
		}
		if profile.FromUser == "" {
			profile.FromUser = "Unknown"
			change = true
		}
		if change {
			st.profiles[name] = profile
		}
	}
	if st.defaultProfile == "" {
		for n := range st.profiles {
			st.defaultProfile = n
		}
	}
	return err
}

func (st *Storage) storeProfiles() error {
	return storeJSON(st.profiles_path, st.profiles)
}

func (st *Storage) migrateToProfile() error {
	st.loadPolicy()
	st.loadConfiguration()
	st.loadDisplayprefs()
	st.loadProfiles()
	st.profiles["Default"] = Profile{
		Policy:        st.policy,
		Configuration: st.configuration,
		Displayprefs:  st.displayprefs,
	}
	return st.storeProfiles()
}

func loadJSON(path string, obj interface{}) error {
	var file []byte
	var err error
	file, err = os.ReadFile(path)
	if err != nil {
		file = []byte("{}")
	}
	err = json.Unmarshal(file, &obj)
	if err != nil {
		log.Printf("ERROR: Failed to read \"%s\": %s", path, err)
	}
	return err
}

func storeJSON(path string, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Printf("ERROR: Failed to write to \"%s\": %s", path, err)
	}
	return err
}
