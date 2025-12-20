package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hrfee/mediabrowser"
)

const (
	// ActivityLimit is the maximum number of ActivityLogEntries to keep in memory.
	// The array they are stored in is fixed, so (ActivityLimit*unsafe.Sizeof(mediabrowser.ActivityLogEntry))
	// At writing ActivityLogEntries take up ~160 bytes each, so 1M of memory gives us room for ~6250 records
	ActivityLimit int = 1e6 / 160
	// If ByUserLimitLength is true, ByUserLengthOrBaseLength is the maximum number of records attached
	// to a user.
	// If false, it is the base amount of entries to allocate for for each user ID, and more will be allocated as needed.
	ByUserLengthOrBaseLength = 128
	ByUserLimitLength        = false
)

type activityLogEntrySource interface {
	GetActivityLog(skip, limit int, since time.Time, hasUserID bool) (mediabrowser.ActivityLog, error)
}

// JFActivityCache is a cache for Jellyfin ActivityLogEntries, intended to be refreshed frequently
// and suited to it by only querying for changes since the last refresh.
type JFActivityCache struct {
	jf    activityLogEntrySource
	cache [ActivityLimit]mediabrowser.ActivityLogEntry
	// index into Cache of the entry that should be considered the start (i.e. most recent), and end (i.e. oldest).
	start, end int
	// Map of activity entry IDs to their index.
	byEntryID map[int64]int
	// Map of user IDs to a slice of entry indexes they are referenced in, chronologically ordered.
	byUserID                   map[string][]int
	LastSync, LastYieldingSync time.Time
	// Age of cache before it should be refreshed.
	WaitForSyncTimeout time.Duration
	syncLock           sync.Mutex
	syncing            bool
	// Total number of entries.
	Total           int
	dupesInLastSync int
}

func (c *JFActivityCache) debugString() string {
	var b strings.Builder
	places := len(strconv.Itoa(ActivityLimit - 1))
	b.Grow((ActivityLimit * (places + 1) * 2) + 1)
	for i := range c.cache {
		fmt.Fprintf(&b, "%0"+strconv.Itoa(places)+"d|", i)
	}
	b.WriteByte('\n')
	for i := range c.cache {
		fmt.Fprintf(&b, "%0"+strconv.Itoa(places)+"d|", c.cache[i].ID)
	}
	return b.String()
}

// NewJFActivityCache returns a Jellyfin ActivityLogEntry cache.
// You should set the timeout low, as events are likely to happen frequently,
// and refreshing should be quick anyway
func NewJFActivityCache(jf activityLogEntrySource, waitForSyncTimeout time.Duration) *JFActivityCache {
	c := &JFActivityCache{
		jf:                 jf,
		WaitForSyncTimeout: waitForSyncTimeout,
		start:              -1,
		end:                -1,
		byEntryID:          map[int64]int{},
		byUserID:           map[string][]int{},
		Total:              0,
		dupesInLastSync:    0,
	}
	for i := range ActivityLimit {
		c.cache[i].ID = -1
	}
	return c
}

// ByUserID returns a slice of ActivitLogEntries with the given jellyfin ID attached.
func (c *JFActivityCache) ByUserID(jellyfinID string) ([]mediabrowser.ActivityLogEntry, error) {
	if err := c.MaybeSync(); err != nil {
		return nil, err
	}
	arr, ok := c.byUserID[jellyfinID]
	if !ok {
		return nil, nil
	}
	out := make([]mediabrowser.ActivityLogEntry, len(arr))
	for i, aleIdx := range arr {
		out[i] = c.cache[aleIdx]
	}
	return out, nil
}

// ByEntryID returns the ActivityLogEntry with the corresponding ID.
func (c *JFActivityCache) ByEntryID(entryID int64) (entry mediabrowser.ActivityLogEntry, ok bool, err error) {
	err = c.MaybeSync()
	if err != nil {
		return
	}
	var idx int
	idx, ok = c.byEntryID[entryID]
	if !ok {
		return
	}
	entry = c.cache[idx]
	return
}

// MaybeSync returns once the cache is in a suitable state to read:
// return if cache is fresh, sync if not, or wait if another sync is happening already.
func (c *JFActivityCache) MaybeSync() error {
	shouldWaitForSync := time.Now().After(c.LastSync.Add(c.WaitForSyncTimeout))

	if !shouldWaitForSync {
		return nil
	}

	syncStatus := make(chan error)

	go func(status chan error, c *JFActivityCache) {
		c.syncLock.Lock()
		alreadySyncing := c.syncing
		// We're either already syncing or will be
		c.syncing = true
		c.syncLock.Unlock()
		if !alreadySyncing {
			// If we haven't synced, this'll just get max (ActivityLimit),
			// If we have, it'll get anything that's happened since then
			thisSync := time.Now()
			al, err := c.jf.GetActivityLog(-1, ActivityLimit, c.LastYieldingSync, true)
			if err != nil {
				c.syncLock.Lock()
				c.syncing = false
				c.syncLock.Unlock()
				status <- err
				return
			}

			// Can't trust the source fully, so we need to check for anything we've already got stored
			// -before- we decide where the data should go.
			recvLength := len(al.Items)
			c.dupesInLastSync = 0
			for i, ale := range al.Items {
				if _, ok := c.byEntryID[ale.ID]; ok {
					c.dupesInLastSync = len(al.Items) - i
					// If we got the same as before, everything after it we'll also have.
					recvLength = i
					break
				}
			}
			if recvLength > 0 {
				// Lazy strategy: rebuild user ID maps each time.
				// Wipe them, and then append each new refresh element as we process them.
				// Then loop through all the old entries and append them too.
				for uid := range c.byUserID {
					c.byUserID[uid] = c.byUserID[uid][:0]
				}

				previousStart := c.start

				if c.start == -1 {
					c.start = 0
					c.end = recvLength - 1
				} else {
					c.start = ((c.start-recvLength)%ActivityLimit + ActivityLimit) % ActivityLimit
				}
				if c.cache[c.start].ID != -1 {
					c.end = ((c.end-1)%ActivityLimit + ActivityLimit) % ActivityLimit
				}
				for i := range recvLength {
					ale := al.Items[i]
					ci := (c.start + i) % ActivityLimit
					if c.cache[ci].ID != -1 {
						// Since we're overwriting it, remove it from index
						delete(c.byEntryID, c.cache[ci].ID)
						// don't increment total since we're adding and removing
					} else {
						c.Total++
					}
					if ale.UserID != "" {
						arr, ok := c.byUserID[ale.UserID]
						if !ok {
							arr = make([]int, 0, ByUserLengthOrBaseLength)
						}
						if !ByUserLimitLength || len(arr) < ByUserLengthOrBaseLength {
							arr = append(arr, ci)
							c.byUserID[ale.UserID] = arr
						}
					}

					c.cache[ci] = ale
					c.byEntryID[ale.ID] = ci
				}
				// If this was the first sync, everything has already been processed in the previous loop.
				if previousStart != -1 {
					i := previousStart
					for {
						if c.cache[i].UserID != "" {
							arr, ok := c.byUserID[c.cache[i].UserID]
							if !ok {
								arr = make([]int, 0, ByUserLengthOrBaseLength)
							}
							if !ByUserLimitLength || len(arr) < ByUserLengthOrBaseLength {
								arr = append(arr, i)
								c.byUserID[c.cache[i].UserID] = arr
							}
						}

						if i == c.end {
							break
						}
						i = (i + 1) % ActivityLimit
					}
				}
			}

			// for i := range c.cache {
			// 	fmt.Printf("%04d|", i)
			// }
			// fmt.Print("\n")
			// for i := range c.cache {
			// 	fmt.Printf("%04d|", c.cache[i].ID)
			// }
			// fmt.Print("\n")

			c.syncLock.Lock()
			c.LastSync = thisSync
			if recvLength > 0 {
				c.LastYieldingSync = thisSync
			}
			c.syncing = false
			c.syncLock.Unlock()
		} else {
			for c.syncing {
				continue
			}
		}
		status <- nil
	}(syncStatus, c)
	err := <-syncStatus
	return err
}
