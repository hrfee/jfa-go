package main

import (
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

func (c *UserCache) Gen(app *appContext) error {
	if !time.Now().After(c.LastSync.Add(WEB_USER_CACHE_SYNC)) {
		return nil
	}
	users, err := app.jf.GetUsers(false)
	if err != nil {
		return err
	}
	c.Lock.Lock()
	c.Cache = make([]respUser, len(users))
	for i, jfUser := range users {
		c.Cache[i] = app.userSummary(jfUser)
	}
	c.LastSync = time.Now()
	c.Lock.Unlock()
	return nil
}
