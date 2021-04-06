package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hrfee/mediabrowser"
)

type Storage struct {
	timePattern                                                                                                                           string
	invite_path, emails_path, policy_path, configuration_path, displayprefs_path, ombi_path, profiles_path, customEmails_path, users_path string
	users                                                                                                                                 map[string]time.Time
	invites                                                                                                                               Invites
	profiles                                                                                                                              map[string]Profile
	defaultProfile                                                                                                                        string
	emails, displayprefs, ombi_template                                                                                                   map[string]interface{}
	customEmails                                                                                                                          customEmails
	policy                                                                                                                                mediabrowser.Policy
	configuration                                                                                                                         mediabrowser.Configuration
	lang                                                                                                                                  Lang
	invitesLock, usersLock                                                                                                                sync.Mutex
}

type customEmails struct {
	UserCreated       customEmail `json:"userCreated"`
	InviteExpiry      customEmail `json:"inviteExpiry"`
	PasswordReset     customEmail `json:"passwordReset"`
	UserDeleted       customEmail `json:"userDeleted"`
	InviteEmail       customEmail `json:"inviteEmail"`
	WelcomeEmail      customEmail `json:"welcomeEmail"`
	EmailConfirmation customEmail `json:"emailConfirmation"`
	UserExpired       customEmail `json:"userExpired"`
}

type customEmail struct {
	Enabled   bool     `json:"enabled,omitempty"`
	Content   string   `json:"content"`
	Variables []string `json:"variables,omitempty"`
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
}

type Invite struct {
	Created       time.Time                  `json:"created"`
	NoLimit       bool                       `json:"no-limit"`
	RemainingUses int                        `json:"remaining-uses"`
	ValidTill     time.Time                  `json:"valid_till"`
	UserExpiry    bool                       `json:"user-duration"`
	UserDays      int                        `json:"user-days,omitempty"`
	UserHours     int                        `json:"user-hours,omitempty"`
	UserMinutes   int                        `json:"user-minutes,omitempty"`
	Email         string                     `json:"email"`
	UsedBy        [][]string                 `json:"used-by"`
	Notify        map[string]map[string]bool `json:"notify"`
	Profile       string                     `json:"profile"`
	Label         string                     `json:"label,omitempty"`
	Keys          []string                   `json:"keys,omitempty"`
}

type Lang struct {
	AdminPath         string
	chosenAdminLang   string
	Admin             adminLangs
	AdminJSON         map[string]string
	FormPath          string
	chosenFormLang    string
	Form              formLangs
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
	err = st.loadLangForm(filesystems...)
	if err != nil {
		return
	}
	err = st.loadLangPWR(filesystems...)
	if err != nil {
		return
	}
	err = st.loadLangEmail(filesystems...)
	return
}

func (common *commonLangs) patchCommon(lang string, other *langSection) {
	if *other == nil {
		*other = langSection{}
	}
	if _, ok := (*common)[lang]; !ok {
		lang = "en-us"
	}
	for n, ev := range (*common)[lang].Strings {
		if v, ok := (*other)[n]; !ok || v == "" {
			(*other)[n] = ev
		}
	}
}

// If a given language has missing values, fill it in with the english value.
func patchLang(english, other *langSection) {
	if *other == nil {
		*other = langSection{}
	}
	for n, ev := range *english {
		if v, ok := (*other)[n]; !ok || v == "" {
			(*other)[n] = ev
		}
	}
}

func patchQuantityStrings(english, other *map[string]quantityString) {
	for n, ev := range *english {
		qs, ok := (*other)[n]
		if !ok {
			(*other)[n] = ev
			return
		} else if qs.Singular == "" {
			qs.Singular = ev.Singular
		} else if (*other)[n].Plural == "" {
			qs.Plural = ev.Plural
		}
		(*other)[n] = qs
	}
}

func (st *Storage) loadLangCommon(filesystems ...fs.FS) error {
	st.lang.Common = map[string]commonLang{}
	var english commonLang
	load := func(filesystem fs.FS, fname string) error {
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
			patchLang(&english.Strings, &lang.Strings)
		}
		st.lang.Common[index] = lang
		return nil
	}
	engFound := false
	var err error
	for _, filesystem := range filesystems {
		err = load(filesystem, "en-us.json")
		if err == nil {
			engFound = true
		}
	}
	if !engFound {
		return err
	}
	english = st.lang.Common["en-us"]
	commonLoaded := false
	for _, filesystem := range filesystems {
		files, err := fs.ReadDir(filesystem, st.lang.CommonPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.Name() != "en-us.json" {
				err = load(filesystem, f.Name())
				if err == nil {
					commonLoaded = true
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
	load := func(filesystem fs.FS, fname string) error {
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
		st.lang.Common.patchCommon(index, &lang.Strings)
		if fname != "en-us.json" {
			patchLang(&english.Strings, &lang.Strings)
			patchLang(&english.Notifications, &lang.Notifications)
			patchQuantityStrings(&english.QuantityStrings, &lang.QuantityStrings)
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
	for _, filesystem := range filesystems {
		err = load(filesystem, "en-us.json")
		if err == nil {
			engFound = true
		}
	}
	if !engFound {
		return err
	}
	english = st.lang.Admin["en-us"]
	adminLoaded := false
	for _, filesystem := range filesystems {
		files, err := fs.ReadDir(filesystem, st.lang.AdminPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.Name() != "en-us.json" {
				err = load(filesystem, f.Name())
				if err == nil {
					adminLoaded = true
				}
			}
		}
	}
	if !adminLoaded {
		return err
	}
	return nil
}

func (st *Storage) loadLangForm(filesystems ...fs.FS) error {
	st.lang.Form = map[string]formLang{}
	var english formLang
	load := func(filesystem fs.FS, fname string) error {
		index := strings.TrimSuffix(fname, filepath.Ext(fname))
		lang := formLang{}
		f, err := fs.ReadFile(filesystem, FSJoin(st.lang.FormPath, fname))
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
		st.lang.Common.patchCommon(index, &lang.Strings)
		if fname != "en-us.json" {
			patchLang(&english.Strings, &lang.Strings)
			patchLang(&english.Notifications, &lang.Notifications)
			patchQuantityStrings(&english.ValidationStrings, &lang.ValidationStrings)
		}
		notifications, err := json.Marshal(lang.Notifications)
		if err != nil {
			return err
		}
		validationStrings, err := json.Marshal(lang.ValidationStrings)
		if err != nil {
			return err
		}
		lang.notificationsJSON = string(notifications)
		lang.validationStringsJSON = string(validationStrings)
		st.lang.Form[index] = lang
		return nil
	}
	engFound := false
	var err error
	for _, filesystem := range filesystems {
		err = load(filesystem, "en-us.json")
		if err == nil {
			engFound = true
		}
	}
	if !engFound {
		return err
	}
	english = st.lang.Form["en-us"]
	formLoaded := false
	for _, filesystem := range filesystems {
		files, err := fs.ReadDir(filesystem, st.lang.FormPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.Name() != "en-us.json" {
				err = load(filesystem, f.Name())
				if err == nil {
					formLoaded = true
				}
			}
		}
	}
	if !formLoaded {
		return err
	}
	return nil
}

func (st *Storage) loadLangPWR(filesystems ...fs.FS) error {
	st.lang.PasswordReset = map[string]pwrLang{}
	var english pwrLang
	load := func(filesystem fs.FS, fname string) error {
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
		st.lang.Common.patchCommon(index, &lang.Strings)
		if fname != "en-us.json" {
			patchLang(&english.Strings, &lang.Strings)
		}
		st.lang.PasswordReset[index] = lang
		return nil
	}
	engFound := false
	var err error
	for _, filesystem := range filesystems {
		err = load(filesystem, "en-us.json")
		if err == nil {
			engFound = true
		}
	}
	if !engFound {
		return err
	}
	english = st.lang.PasswordReset["en-us"]
	formLoaded := false
	for _, filesystem := range filesystems {
		files, err := fs.ReadDir(filesystem, st.lang.PasswordResetPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.Name() != "en-us.json" {
				err = load(filesystem, f.Name())
				if err == nil {
					formLoaded = true
				}
			}
		}
	}
	if !formLoaded {
		return err
	}
	return nil
}

func (st *Storage) loadLangEmail(filesystems ...fs.FS) error {
	st.lang.Email = map[string]emailLang{}
	var english emailLang
	load := func(filesystem fs.FS, fname string) error {
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
		st.lang.Common.patchCommon(index, &lang.Strings)
		if fname != "en-us.json" {
			patchLang(&english.UserCreated, &lang.UserCreated)
			patchLang(&english.InviteExpiry, &lang.InviteExpiry)
			patchLang(&english.PasswordReset, &lang.PasswordReset)
			patchLang(&english.UserDeleted, &lang.UserDeleted)
			patchLang(&english.InviteEmail, &lang.InviteEmail)
			patchLang(&english.WelcomeEmail, &lang.WelcomeEmail)
		}
		st.lang.Email[index] = lang
		return nil
	}
	engFound := false
	var err error
	for _, filesystem := range filesystems {
		err = load(filesystem, "en-us.json")
		if err == nil {
			engFound = true
		}
	}
	if !engFound {
		return err
	}
	english = st.lang.Email["en-us"]
	emailLoaded := false
	for _, filesystem := range filesystems {
		files, err := fs.ReadDir(filesystem, st.lang.EmailPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.Name() != "en-us.json" {
				err = load(filesystem, f.Name())
				if err == nil {
					emailLoaded = true
				}
			}
		}
	}
	if !emailLoaded {
		return err
	}
	return nil
}

type Invites map[string]Invite

func (st *Storage) loadInvites() error {
	st.invitesLock.Lock()
	defer st.invitesLock.Unlock()
	return loadJSON(st.invite_path, &st.invites)
}

func (st *Storage) storeInvites() error {
	st.invitesLock.Lock()
	defer st.invitesLock.Unlock()
	return storeJSON(st.invite_path, st.invites)
}

func (st *Storage) loadUsers() error {
	st.usersLock.Lock()
	defer st.usersLock.Unlock()
	if st.users == nil {
		st.users = map[string]time.Time{}
	}
	temp := map[string]time.Time{}
	err := loadJSON(st.users_path, &temp)
	if err != nil {
		return err
	}
	for id, t1 := range temp {
		if _, ok := st.users[id]; !ok {
			st.users[id] = t1
		}
	}
	fmt.Printf("CURRENT USERS:\n%+v\n", st.users)
	return nil
}

func (st *Storage) storeUsers() error {
	return storeJSON(st.users_path, st.users)
}

func (st *Storage) loadEmails() error {
	return loadJSON(st.emails_path, &st.emails)
}

func (st *Storage) storeEmails() error {
	return storeJSON(st.emails_path, st.emails)
}

func (st *Storage) loadCustomEmails() error {
	return loadJSON(st.customEmails_path, &st.customEmails)
}

func (st *Storage) storeCustomEmails() error {
	return storeJSON(st.customEmails_path, st.customEmails)
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

// One build of JF 10.7.0 hyphenated user IDs while another one later didn't. These functions will hyphenate/de-hyphenate email storage.

func hyphenate(userID string) string {
	if userID[8] == '-' {
		return userID
	}
	return userID[:8] + "-" + userID[8:12] + "-" + userID[12:16] + "-" + userID[16:20] + "-" + userID[20:]
}

func (app *appContext) deHyphenateStorage(old map[string]interface{}) (map[string]interface{}, int, error) {
	jfUsers, status, err := app.jf.GetUsers(false)
	if status != 200 || err != nil {
		return nil, status, err
	}
	newEmails := map[string]interface{}{}
	for _, user := range jfUsers {
		unHyphenated := user.ID
		hyphenated := hyphenate(unHyphenated)
		val, ok := old[hyphenated]
		if ok {
			newEmails[unHyphenated] = val
		}
	}
	return newEmails, status, err
}

func (app *appContext) hyphenateStorage(old map[string]interface{}) (map[string]interface{}, int, error) {
	jfUsers, status, err := app.jf.GetUsers(false)
	if status != 200 || err != nil {
		return nil, status, err
	}
	newEmails := map[string]interface{}{}
	for _, user := range jfUsers {
		unstripped := user.ID
		stripped := strings.ReplaceAll(unstripped, "-", "")
		val, ok := old[stripped]
		if ok {
			newEmails[unstripped] = val
		}
	}
	return newEmails, status, err
}

func (app *appContext) hyphenateEmailStorage(old map[string]interface{}) (map[string]interface{}, int, error) {
	return app.hyphenateStorage(old)
}

func (app *appContext) deHyphenateEmailStorage(old map[string]interface{}) (map[string]interface{}, int, error) {
	return app.deHyphenateStorage(old)
}

func (app *appContext) hyphenateUserStorage(old map[string]time.Time) (map[string]time.Time, int, error) {
	asInterface := map[string]interface{}{}
	for k, v := range old {
		asInterface[k] = v
	}
	fixed, status, err := app.hyphenateStorage(asInterface)
	if err != nil {
		return nil, status, err
	}
	out := map[string]time.Time{}
	for k, v := range fixed {
		out[k] = v.(time.Time)
	}
	return out, status, err
}

func (app *appContext) deHyphenateUserStorage(old map[string]time.Time) (map[string]time.Time, int, error) {
	asInterface := map[string]interface{}{}
	for k, v := range old {
		asInterface[k] = v
	}
	fixed, status, err := app.deHyphenateStorage(asInterface)
	if err != nil {
		return nil, status, err
	}
	out := map[string]time.Time{}
	for k, v := range fixed {
		out[k] = v.(time.Time)
	}
	return out, status, err
}
