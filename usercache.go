package main

import (
	"cmp"
	"sync"
	"time"
)

const (
	// FIXME: Follow mediabrowser, or make tuneable, or both
	WEB_USER_CACHE_SYNC = 30 * time.Second
)

type UserCache struct {
	Cache    []respUser
	LastSync time.Time
	Lock     sync.Mutex
}

func (c *UserCache) Gen(app *appContext) ([]respUser, error) {
	// FIXME: I don't like this.
	if !time.Now().After(c.LastSync.Add(WEB_USER_CACHE_SYNC)) {
		return c.Cache, nil
	}
	c.Lock.Lock()
	users, err := app.jf.GetUsers(false)
	if err != nil {
		return nil, err
	}
	c.Cache = make([]respUser, len(users))
	for i, jfUser := range users {
		c.Cache[i] = app.userSummary(jfUser)
	}
	c.LastSync = time.Now()
	c.Lock.Unlock()
	return c.Cache, nil
}

type Less func(a, b *respUser) bool
type SortableUserList struct {
	Cache    []respUser
	lessFunc Less
}

func (sc *SortableUserList) Len() int {
	return len(sc.Cache)
}

func (sc *SortableUserList) Swap(i, j int) {
	sc.Cache[i], sc.Cache[j] = sc.Cache[j], sc.Cache[i]
}

func (sc *SortableUserList) Less(i, j int) bool {
	return sc.lessFunc(&sc.Cache[i], &sc.Cache[j])
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

// Allow sorting by respUser's struct fields (well, it's JSON-representation's fields)
// Ugly I know, but at least cmp.Less exists.
// Done with vim macros, thank god they exist
func SortUsersBy(u []respUser, field string) SortableUserList {
	s := SortableUserList{Cache: u}

	switch field {
	case "id":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.ID, b.ID)
		}

	case "name":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.Name, b.Name)
		}
	case "email":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.Email, b.Email)
		}
	case "notify_email":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(bool2int(a.NotifyThroughEmail), bool2int(b.NotifyThroughEmail))
		}
	case "last_active":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.LastActive, b.LastActive)
		}
	case "admin":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(bool2int(a.Admin), bool2int(b.Admin))
		}
	case "expiry":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.Expiry, b.Expiry)
		}
	case "disabled":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(bool2int(a.Disabled), bool2int(b.Disabled))
		}
	case "telegram":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.Telegram, b.Telegram)
		}
	case "notify_telegram":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(bool2int(a.NotifyThroughTelegram), bool2int(b.NotifyThroughTelegram))
		}
	case "discord":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.Discord, b.Discord)
		}
	case "discord_id":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.DiscordID, b.DiscordID)
		}
	case "notify_discord":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(bool2int(a.NotifyThroughDiscord), bool2int(b.NotifyThroughDiscord))
		}
	case "matrix":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.Matrix, b.Matrix)
		}
	case "notify_matrix":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(bool2int(a.NotifyThroughMatrix), bool2int(b.NotifyThroughMatrix))
		}
	case "label":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(a.Label, b.Label)
		}
	case "accounts_admin":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(bool2int(a.AccountsAdmin), bool2int(b.AccountsAdmin))
		}
	case "referrals_enabled":
		s.lessFunc = func(a, b *respUser) bool {
			return cmp.Less(bool2int(a.ReferralsEnabled), bool2int(b.ReferralsEnabled))
		}
	}
	return s
}

type Filter func(yield func(*respUser) bool)

type FilterableList struct {
	Cache      []respUser
	filterFunc Filter
}
