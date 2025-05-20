package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// FIXME: Follow mediabrowser, or make tuneable, or both
	WEB_USER_CACHE_SYNC     = 30 * time.Second
	USER_DEFAULT_SORT_FIELD = "name"
)

type UserCache struct {
	Cache    []respUser
	Ref      []*respUser
	Sorted   bool
	LastSync time.Time
	Lock     sync.Mutex
}

func (c *UserCache) gen(app *appContext) error {
	// FIXME: I don't like this.
	if !time.Now().After(c.LastSync.Add(WEB_USER_CACHE_SYNC)) {
		return nil
	}
	c.Lock.Lock()
	users, err := app.jf.GetUsers(false)
	if err != nil {
		return err
	}
	c.Cache = make([]respUser, len(users))
	for i, jfUser := range users {
		c.Cache[i] = app.userSummary(jfUser)
	}
	c.Ref = make([]*respUser, len(c.Cache))
	for i := range c.Cache {
		c.Ref[i] = &(c.Cache[i])
	}
	c.Sorted = false
	c.LastSync = time.Now()
	c.Lock.Unlock()
	return nil
}

func (c *UserCache) Gen(app *appContext, sorted bool) ([]*respUser, error) {
	if err := c.gen(app); err != nil {
		return nil, err
	}
	if sorted && !c.Sorted {
		c.Lock.Lock()
		// FIXME: Check we want ascending!
		c.Sort(c.Ref, USER_DEFAULT_SORT_FIELD, true)
		c.Sorted = true
		c.Lock.Unlock()
	}
	return c.Ref, nil
}

// instead of making a Less for bools, just convert them to integers
// https://0x0f.me/blog/golang-compiler-optimization/
func bool2int(b bool) int {
	var i int
	if b {
		i = 1
	} else {
		i = 0
	}
	return i
}

// Returns -1 if respUser < value, 0 if equal, 1 is greater than
type Sorter func(a, b *respUser) int

// Allow sorting by respUser's struct fields (well, it's JSON-representation's fields)
func SortUsersBy(field string) Sorter {
	switch field {
	case "id":
		return func(a, b *respUser) int {
			return cmp.Compare(strings.ToLower(a.ID), strings.ToLower(b.ID))
		}
	case "name":
		return func(a, b *respUser) int {
			return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		}
	case "email":
		return func(a, b *respUser) int {
			return cmp.Compare(strings.ToLower(a.Email), strings.ToLower(b.Email))
		}
	case "notify_email":
		return func(a, b *respUser) int {
			return cmp.Compare(bool2int(a.NotifyThroughEmail), bool2int(b.NotifyThroughEmail))
		}
	case "last_active":
		return func(a, b *respUser) int {
			return cmp.Compare(a.LastActive, b.LastActive)
		}
	case "admin":
		return func(a, b *respUser) int {
			return cmp.Compare(bool2int(a.Admin), bool2int(b.Admin))
		}
	case "expiry":
		return func(a, b *respUser) int {
			return cmp.Compare(a.Expiry, b.Expiry)
		}
	case "disabled":
		return func(a, b *respUser) int {
			return cmp.Compare(bool2int(a.Disabled), bool2int(b.Disabled))
		}
	case "telegram":
		return func(a, b *respUser) int {
			return cmp.Compare(strings.ToLower(a.Telegram), strings.ToLower(b.Telegram))
		}
	case "notify_telegram":
		return func(a, b *respUser) int {
			return cmp.Compare(bool2int(a.NotifyThroughTelegram), bool2int(b.NotifyThroughTelegram))
		}
	case "discord":
		return func(a, b *respUser) int {
			return cmp.Compare(strings.ToLower(a.Discord), strings.ToLower(b.Discord))
		}
	case "discord_id":
		return func(a, b *respUser) int {
			return cmp.Compare(strings.ToLower(a.DiscordID), strings.ToLower(b.DiscordID))
		}
	case "notify_discord":
		return func(a, b *respUser) int {
			return cmp.Compare(bool2int(a.NotifyThroughDiscord), bool2int(b.NotifyThroughDiscord))
		}
	case "matrix":
		return func(a, b *respUser) int {
			return cmp.Compare(strings.ToLower(a.Matrix), strings.ToLower(b.Matrix))
		}
	case "notify_matrix":
		return func(a, b *respUser) int {
			return cmp.Compare(bool2int(a.NotifyThroughMatrix), bool2int(b.NotifyThroughMatrix))
		}
	case "label":
		return func(a, b *respUser) int {
			return cmp.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
		}
	case "accounts_admin":
		return func(a, b *respUser) int {
			return cmp.Compare(bool2int(a.AccountsAdmin), bool2int(b.AccountsAdmin))
		}
	case "referrals_enabled":
		return func(a, b *respUser) int {
			return cmp.Compare(bool2int(a.ReferralsEnabled), bool2int(b.ReferralsEnabled))
		}
	}
	panic(fmt.Errorf("got invalid field %s", field))
	return nil
}

type Filter func(*respUser) bool

type CompareResult int

const (
	Lesser  CompareResult = -1
	Equal   CompareResult = 0
	Greater CompareResult = 1
)

// One day i'll figure out Go generics
/*type FilterValue interface {
	bool | string | DateAttempt
}*/

type DateAttempt struct {
	Year   *int `json:"year,omitempty"`
	Month  *int `json:"month,omitempty"`
	Day    *int `json:"day,omitempty"`
	Hour   *int `json:"hour,omitempty"`
	Minute *int `json:"minute,omitempty"`
}

// Compares a Unix timestamp.
// We want to compare only the fields given in DateAttempt,
// so we copy subjectDate and apply on those fields from this._value.
func (d DateAttempt) Compare(subject int64) int {
	subjectTime := time.Unix(subject, 0)
	yy, mo, dd := subjectTime.Date()
	hh, mm, _ := subjectTime.Clock()
	if d.Year != nil {
		yy = *d.Year
	}
	if d.Month != nil {
		// Month in Javascript is zero-based, so we need to increment it
		mo = time.Month((*d.Month) + 1)
	}
	if d.Day != nil {
		dd = *d.Day
	}
	if d.Hour != nil {
		hh = *d.Hour
	}
	if d.Minute != nil {
		mm = *d.Minute
	}
	return subjectTime.Compare(time.Date(yy, mo, dd, hh, mm, 0, 0, nil))
}

// FIXME: Consider using QueryDTO.Class rather than assuming type from name? Probably not worthwhile though.
func FilterUsersBy(field string, op QueryOperator, value any) Filter {
	operator := Equal
	switch op {
	case LesserOperator:
		operator = Lesser
	case EqualOperator:
		operator = Equal
	case GreaterOperator:
		operator = Greater
	}

	switch field {
	case "id":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.ID), strings.ToLower(value.(string))) == int(operator)
		}
	case "name":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(value.(string))) == int(operator)
		}
	case "email":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Email), strings.ToLower(value.(string))) == int(operator)
		}
	case "notify_email":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.NotifyThroughEmail), bool2int(value.(bool))) == int(operator)
		}
	case "last_active":
		return func(a *respUser) bool {
			return value.(DateAttempt).Compare(a.LastActive) == int(operator)
		}
	case "admin":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.Admin), bool2int(value.(bool))) == int(operator)
		}
	case "expiry":
		return func(a *respUser) bool {
			return value.(DateAttempt).Compare(a.Expiry) == int(operator)
		}
	case "disabled":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.Disabled), bool2int(value.(bool))) == int(operator)
		}
	case "telegram":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Telegram), strings.ToLower(value.(string))) == int(operator)
		}
	case "notify_telegram":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.NotifyThroughTelegram), bool2int(value.(bool))) == int(operator)
		}
	case "discord":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Discord), strings.ToLower(value.(string))) == int(operator)
		}
	case "discord_id":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.DiscordID), strings.ToLower(value.(string))) == int(operator)
		}
	case "notify_discord":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.NotifyThroughDiscord), bool2int(value.(bool))) == int(operator)
		}
	case "matrix":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Matrix), strings.ToLower(value.(string))) == int(operator)
		}
	case "notify_matrix":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.NotifyThroughMatrix), bool2int(value.(bool))) == int(operator)
		}
	case "label":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Label), strings.ToLower(value.(string))) == int(operator)
		}
	case "accounts_admin":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.AccountsAdmin), bool2int(value.(bool))) == int(operator)
		}
	case "referrals_enabled":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.ReferralsEnabled), bool2int(value.(bool))) == int(operator)
		}
	}
	panic(fmt.Errorf("got invalid field %s", field))
	return nil
}

func (ru *respUser) MatchesSearch(term string) bool {
	return (strings.Contains(ru.ID, term) ||
		strings.Contains(strings.ToLower(ru.Name), term) ||
		strings.Contains(strings.ToLower(ru.Label), term) ||
		strings.Contains(strings.ToLower(ru.Email), term) ||
		strings.Contains(strings.ToLower(ru.Discord), term) ||
		strings.Contains(strings.ToLower(ru.Matrix), term) ||
		strings.Contains(strings.ToLower(ru.Telegram), term))
}

type QueryClass string

const (
	BoolQuery   QueryClass = "bool"
	StringQuery QueryClass = "string"
	DateQuery   QueryClass = "date"
)

type QueryOperator string

const (
	LesserOperator  QueryOperator = "<"
	EqualOperator   QueryOperator = "="
	GreaterOperator QueryOperator = ">"
)

type QueryDTO struct {
	Class    QueryClass    `json:"class"`
	Field    string        `json:"field"`
	Operator QueryOperator `json:"operator"`
	// string | bool | DateAttempt
	Value any `json:"value"`
}

type ServerSearchReqDTO struct {
	PaginatedReqDTO
	SearchTerms []string   `json:"searchTerms"`
	Queries     []QueryDTO `json:"queries"`
}

// Filter by AND-ing all search terms and queries.
func (c *UserCache) Filter(users []*respUser, terms []string, queries []QueryDTO) []*respUser {
	filters := make([]Filter, len(queries))
	for i, q := range queries {
		filters[i] = FilterUsersBy(q.Field, q.Operator, q.Value)
	}
	// FIXME: Properly consider pre-allocation size
	out := make([]*respUser, 0, len(users)/4)
	for i := range users {
		match := true
		for _, term := range terms {
			if !users[i].MatchesSearch(term) {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		for _, filter := range filters {
			if filter == nil || !filter(users[i]) {
				match = false
				break
			}
		}
		if match {
			out = append(out, users[i])
		}
	}
	return out
}

func (c *UserCache) Sort(users []*respUser, field string, ascending bool) {
	slices.SortFunc(users, SortUsersBy(field))
	if !ascending {
		slices.Reverse(users)
	}
}
