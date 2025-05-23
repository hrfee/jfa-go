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
	USER_DEFAULT_SORT_FIELD     = "name"
	USER_DEFAULT_SORT_ASCENDING = true
)

// UserCache caches the transport representation of users,
// complementing the built-in cache of the mediabrowser package.
// Synchronisation runs in the background and consumers receive
// old data for responsiveness unless an extended expiry time has passed.
// It also provides methods for sorting, searching and filtering server-side.
type UserCache struct {
	Cache    []respUser
	Ref      []*respUser
	Sorted   bool
	LastSync time.Time
	// After cache is this old, re-sync, but do it in the background and return the old cache.
	SyncTimeout time.Duration
	// After cache is this old, re-sync and wait for it and return the new cache.
	WaitForSyncTimeout time.Duration
	SyncLock           sync.Mutex
	Syncing            bool
	SortLock           sync.Mutex
	Sorting            bool
}

func NewUserCache(syncTimeout, waitForSyncTimeout time.Duration) *UserCache {
	return &UserCache{
		SyncTimeout:        syncTimeout,
		WaitForSyncTimeout: waitForSyncTimeout,
	}
}

// MaybeSync (maybe) syncs the cache, resulting in updated UserCache.Cache/.Ref/.Sorted.
// Only syncs if c.SyncTimeout duration has passed since last one.
// If c.WaitForSyncTimeout duration has passed, this will block until a sync is complete, otherwise it will sync in the background
// (expecting you to use the old cache data). Only one sync will run at a time.
func (c *UserCache) MaybeSync(app *appContext) error {
	shouldWaitForSync := time.Now().After(c.LastSync.Add(c.WaitForSyncTimeout)) || c.Ref == nil || len(c.Ref) == 0
	shouldSync := time.Now().After(c.LastSync.Add(c.SyncTimeout))

	if !shouldSync {
		return nil
	}

	syncStatus := make(chan error)

	go func(status chan error, c *UserCache) {
		c.SyncLock.Lock()
		alreadySyncing := c.Syncing
		// We're either already syncing or will be
		c.Syncing = true
		c.SyncLock.Unlock()
		if !alreadySyncing {
			users, err := app.jf.GetUsers(false)
			if err != nil {
				c.SyncLock.Lock()
				c.Syncing = false
				c.SyncLock.Unlock()
				status <- err
				return
			}
			cache := make([]respUser, len(users))
			for i, jfUser := range users {
				cache[i] = app.userSummary(jfUser)
			}
			ref := make([]*respUser, len(cache))
			for i := range cache {
				ref[i] = &(cache[i])
			}
			c.Cache = cache
			c.Ref = ref
			c.Sorted = false
			c.LastSync = time.Now()

			c.SyncLock.Lock()
			c.Syncing = false
			c.SyncLock.Unlock()
		} else {
			for c.Syncing {
				continue
			}
		}
		status <- nil
	}(syncStatus, c)

	if shouldWaitForSync {
		err := <-syncStatus
		return err
	}
	return nil
}

func (c *UserCache) GetUserDTOs(app *appContext, sorted bool) ([]*respUser, error) {
	if err := c.MaybeSync(app); err != nil {
		return nil, err
	}
	if sorted && !c.Sorted {
		c.SortLock.Lock()
		alreadySorting := c.Sorting
		c.Sorting = true
		c.SortLock.Unlock()
		if !alreadySorting {
			c.Sort(c.Ref, USER_DEFAULT_SORT_FIELD, USER_DEFAULT_SORT_ASCENDING)
			c.Sorted = true
			c.SortLock.Lock()
			c.Sorting = false
			c.SortLock.Unlock()
		} else {
			for c.Sorting {
				continue
			}
		}
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

// Sorter compares the given field of two respUsers, returning -1 if a < b, 0 if a == b, 1 if a > b.
type Sorter func(a, b *respUser) int

// Allow sorting by respUser's struct fields (well, it's JSON-representation's fields)
// SortUsersBy returns a Sorter function, which compares the given field of two respUsers, returning -1 if a < b, 0 if a == b, 1 if a > b.
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
}

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

// Filter returns true if a specific field in the passed respUser matches some internally defined value.
type Filter func(*respUser) bool

// AsFilter returns a Filter function, which compares the queries value to the corresponding field's value in a passed respUser.
func (q QueryDTO) AsFilter() Filter {
	operator := Equal
	switch q.Operator {
	case LesserOperator:
		operator = Lesser
	case EqualOperator:
		operator = Equal
	case GreaterOperator:
		operator = Greater
	}

	switch q.Field {
	case "id":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.ID), strings.ToLower(q.Value.(string))) == int(operator)
		}
	case "name":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(q.Value.(string))) == int(operator)
		}
	case "email":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Email), strings.ToLower(q.Value.(string))) == int(operator)
		}
	case "notify_email":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.NotifyThroughEmail), bool2int(q.Value.(bool))) == int(operator)
		}
	case "last_active":
		return func(a *respUser) bool {
			return q.Value.(DateAttempt).Compare(a.LastActive) == int(operator)
		}
	case "admin":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.Admin), bool2int(q.Value.(bool))) == int(operator)
		}
	case "expiry":
		return func(a *respUser) bool {
			return q.Value.(DateAttempt).Compare(a.Expiry) == int(operator)
		}
	case "disabled":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.Disabled), bool2int(q.Value.(bool))) == int(operator)
		}
	case "telegram":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Telegram), strings.ToLower(q.Value.(string))) == int(operator)
		}
	case "notify_telegram":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.NotifyThroughTelegram), bool2int(q.Value.(bool))) == int(operator)
		}
	case "discord":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Discord), strings.ToLower(q.Value.(string))) == int(operator)
		}
	case "discord_id":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.DiscordID), strings.ToLower(q.Value.(string))) == int(operator)
		}
	case "notify_discord":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.NotifyThroughDiscord), bool2int(q.Value.(bool))) == int(operator)
		}
	case "matrix":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Matrix), strings.ToLower(q.Value.(string))) == int(operator)
		}
	case "notify_matrix":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.NotifyThroughMatrix), bool2int(q.Value.(bool))) == int(operator)
		}
	case "label":
		return func(a *respUser) bool {
			return cmp.Compare(strings.ToLower(a.Label), strings.ToLower(q.Value.(string))) == int(operator)
		}
	case "accounts_admin":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.AccountsAdmin), bool2int(q.Value.(bool))) == int(operator)
		}
	case "referrals_enabled":
		return func(a *respUser) bool {
			return cmp.Compare(bool2int(a.ReferralsEnabled), bool2int(q.Value.(bool))) == int(operator)
		}
	}
	panic(fmt.Errorf("got invalid q.Field %s", q.Field))
}

// MatchesSearch checks (case-insensitively) if any string field in respUser includes the term string.
func (ru *respUser) MatchesSearch(term string) bool {
	return (strings.Contains(ru.ID, term) ||
		strings.Contains(strings.ToLower(ru.Name), term) ||
		strings.Contains(strings.ToLower(ru.Label), term) ||
		strings.Contains(strings.ToLower(ru.Email), term) ||
		strings.Contains(strings.ToLower(ru.Discord), term) ||
		strings.Contains(strings.ToLower(ru.Matrix), term) ||
		strings.Contains(strings.ToLower(ru.Telegram), term))
}

// QueryClass is the class of a query (the datatype), i.e. bool, string or date.
type QueryClass string

const (
	BoolQuery   QueryClass = "bool"
	StringQuery QueryClass = "string"
	DateQuery   QueryClass = "date"
)

// QueryOperator is the operator used for comparison in a filter, i.e. <, = or >.
type QueryOperator string

const (
	LesserOperator  QueryOperator = "<"
	EqualOperator   QueryOperator = "="
	GreaterOperator QueryOperator = ">"
)

// QueryDTO is the transport representation of a Query, sent from the web app.
type QueryDTO struct {
	Class    QueryClass    `json:"class"`
	Field    string        `json:"field"`
	Operator QueryOperator `json:"operator"`
	// string | bool | DateAttempt
	Value any `json:"value"`
}

// ServerSearchReqDTO is a usual PaginatedReqDTO with added fields for searching and filtering.
type ServerSearchReqDTO struct {
	PaginatedReqDTO
	SearchTerms []string   `json:"searchTerms"`
	Queries     []QueryDTO `json:"queries"`
}

// Filter reduces the passed slice of *respUsers
// by searching for each term of terms[] with respUser.MatchesSearch,
// and by evaluating Queries with Query.AsFilter().
func (c *UserCache) Filter(users []*respUser, terms []string, queries []QueryDTO) []*respUser {
	filters := make([]Filter, len(queries))
	for i, q := range queries {
		filters[i] = q.AsFilter()
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

// Sort sorts the given slice of of *respUsers in-place by the field name given, in ascending or descending order.
func (c *UserCache) Sort(users []*respUser, field string, ascending bool) {
	slices.SortFunc(users, SortUsersBy(field))
	if !ascending {
		slices.Reverse(users)
	}
}
