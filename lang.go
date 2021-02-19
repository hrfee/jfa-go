package main

import (
	"strings"
)

type langMeta struct {
	Name string `json:"name"`
}

type quantityString struct {
	Singular string `json:"singular"`
	Plural   string `json:"plural"`
}

type adminLangs map[string]adminLang

func (ls *adminLangs) getOptions() [][2]string {
	opts := make([][2]string, len(*ls))
	i := 0
	for key, lang := range *ls {
		opts[i] = [2]string{key, lang.Meta.Name}
		i++
	}
	return opts
}

type commonLangs map[string]commonLang

type commonLang struct {
	Meta    langMeta    `json:"meta"`
	Strings langSection `json:"strings"`
}

type adminLang struct {
	Meta            langMeta                  `json:"meta"`
	Strings         langSection               `json:"strings"`
	Notifications   langSection               `json:"notifications"`
	QuantityStrings map[string]quantityString `json:"quantityStrings"`
	JSON            string
}

type formLangs map[string]formLang

func (ls *formLangs) getOptions() [][2]string {
	opts := make([][2]string, len(*ls))
	i := 0
	for key, lang := range *ls {
		opts[i] = [2]string{key, lang.Meta.Name}
		i++
	}
	return opts
}

type formLang struct {
	Meta                  langMeta    `json:"meta"`
	Strings               langSection `json:"strings"`
	Notifications         langSection `json:"notifications"`
	notificationsJSON     string
	ValidationStrings     map[string]quantityString `json:"validationStrings"`
	validationStringsJSON string
}

type emailLangs map[string]emailLang

func (ls *emailLangs) getOptions() [][2]string {
	opts := make([][2]string, len(*ls))
	i := 0
	for key, lang := range *ls {
		opts[i] = [2]string{key, lang.Meta.Name}
		i++
	}
	return opts
}

type emailLang struct {
	Meta              langMeta    `json:"meta"`
	Strings           langSection `json:"strings"`
	UserCreated       langSection `json:"userCreated"`
	InviteExpiry      langSection `json:"inviteExpiry"`
	PasswordReset     langSection `json:"passwordReset"`
	UserDeleted       langSection `json:"userDeleted"`
	InviteEmail       langSection `json:"inviteEmail"`
	WelcomeEmail      langSection `json:"welcomeEmail"`
	EmailConfirmation langSection `json:"emailConfirmation"`
}

type setupLangs map[string]setupLang

type setupLang struct {
	Meta               langMeta    `json:"meta"`
	Strings            langSection `json:"strings"`
	StartPage          langSection `json:"startPage"`
	EndPage            langSection `json:"endPage"`
	General            langSection `json:"general"`
	Language           langSection `json:"language"`
	Login              langSection `json:"login"`
	JellyfinEmby       langSection `json:"jellyfinEmby"`
	Ombi               langSection `json:"ombi"`
	Email              langSection `json:"email"`
	Notifications      langSection `json:"notifications"`
	WelcomeEmails      langSection `json:"welcomeEmails"`
	PasswordResets     langSection `json:"passwordResets"`
	InviteEmails       langSection `json:"inviteEmails"`
	PasswordValidation langSection `json:"passwordValidation"`
	HelpMessages       langSection `json:"helpMessages"`
	JSON               string
}

func (ls *setupLangs) getOptions() [][2]string {
	opts := make([][2]string, len(*ls))
	i := 0
	for key, lang := range *ls {
		opts[i] = [2]string{key, lang.Meta.Name}
		i++
	}
	return opts
}

type langSection map[string]string
type tmpl map[string]string

func (el langSection) template(field string, vals tmpl) string {
	text := el.get(field)
	for key, val := range vals {
		text = strings.ReplaceAll(text, "{"+key+"}", val)
	}
	return text
}

func (el langSection) format(field string, vals ...string) string {
	text := el.get(field)
	for _, val := range vals {
		text = strings.Replace(text, "{n}", val, 1)
	}
	return text
}

func (el langSection) get(field string) string {
	t, ok := el[field]
	if !ok {
		return ""
	}
	return t
}
