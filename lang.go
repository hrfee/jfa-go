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

func (ls *adminLangs) getOptions(chosen string) (string, []string) {
	opts := make([]string, len(*ls))
	chosenLang := (*ls)[chosen].Meta.Name
	i := 0
	for _, lang := range *ls {
		opts[i] = lang.Meta.Name
		i++
	}
	return chosenLang, opts
}

type adminLang struct {
	Meta            langMeta                  `json:"meta"`
	Strings         langSection               `json:"strings"`
	Notifications   langSection               `json:"notifications"`
	QuantityStrings map[string]quantityString `json:"quantityStrings"`
	JSON            string
}

type formLangs map[string]formLang

func (ls *formLangs) getOptions(chosen string) (string, []string) {
	opts := make([]string, len(*ls))
	chosenLang := (*ls)[chosen].Meta.Name
	i := 0
	for _, lang := range *ls {
		opts[i] = lang.Meta.Name
		i++
	}
	return chosenLang, opts
}

type formLang struct {
	Meta                  langMeta                  `json:"meta"`
	Strings               langSection               `json:"strings"`
	ValidationStrings     map[string]quantityString `json:"validationStrings"`
	validationStringsJSON string
}

type emailLangs map[string]emailLang

func (ls *emailLangs) getOptions(chosen string) (string, []string) {
	opts := make([]string, len(*ls))
	chosenLang := (*ls)[chosen].Meta.Name
	i := 0
	for _, lang := range *ls {
		opts[i] = lang.Meta.Name
		i++
	}
	return chosenLang, opts
}

type emailLang struct {
	Meta          langMeta    `json:"meta"`
	UserCreated   langSection `json:"userCreated"`
	InviteExpiry  langSection `json:"inviteExpiry"`
	PasswordReset langSection `json:"passwordReset"`
	UserDeleted   langSection `json:"userDeleted"`
	InviteEmail   langSection `json:"inviteEmail"`
}

type langSection map[string]string

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
