package main

import (
	"sync"
	"testing"
	"time"

	"github.com/hrfee/mediabrowser"
)

type MockActivityLogSource struct {
	logs []mediabrowser.ActivityLogEntry
	lock sync.Mutex
	i    int
}

func (m *MockActivityLogSource) run(size int, delay time.Duration, finished *bool) {
	m.logs = make([]mediabrowser.ActivityLogEntry, size)
	for i := range len(m.logs) {
		m.logs[i].ID = -1
	}
	m.i = 0
	for i := range len(m.logs) {
		m.lock.Lock()
		log := mediabrowser.ActivityLogEntry{
			ID:   int64(i),
			Date: mediabrowser.Time{time.Now()},
		}
		m.logs[i] = log
		m.i = i + 1
		m.lock.Unlock()
		time.Sleep(delay)
	}
	*finished = true
	time.Sleep(delay)
}

func (m *MockActivityLogSource) GetActivityLog(skip, limit int, since time.Time, hasUserID bool) (mediabrowser.ActivityLog, error) {
	// This may introduce duplicates, but those are handled fine.
	// If we don't do this, things go wrong in a way that seems
	// very specific to this test setup, and (imo) is not necessarily
	// applicable to a real scenario.
	// since = since.Add(-time.Millisecond)
	out := make([]mediabrowser.ActivityLogEntry, 0, limit)
	count := 0
	loopCount := 0
	m.lock.Lock()
	for i := m.i - 1; count < limit && i >= 0; i-- {
		loopCount++
		if m.logs[i].Date.After(since) {
			out = append(out, m.logs[i])
			count++
		}
	}
	m.lock.Unlock()
	return mediabrowser.ActivityLog{Items: out}, nil
}

func TestJFActivityLog(t *testing.T) {
	t.Parallel()
	// FIXME: This test is failing
	t.Run("Completeness", func(t *testing.T) {
		mock := MockActivityLogSource{}
		waitForSync := time.Microsecond
		cache := NewJFActivityCache(&mock, waitForSync)
		finished := false
		count := len(cache.cache) - 10
		go mock.run(count, time.Millisecond, &finished)
		for {
			if err := cache.MaybeSync(); err != nil {
				t.Errorf("sync failed: %v", err)
				return
			}

			if cache.dupesInLastSync > 1 {
				t.Logf("got %d dupes in last sync\n", cache.dupesInLastSync)
			}

			if finished {
				// Make sure we got everything
				time.Sleep(30 * waitForSync)
				if err := cache.MaybeSync(); err != nil {
					t.Errorf("sync failed: %v", err)
					return
				}
				break
			}
		}
		t.Log(">-\n" + cache.debugString())
		if cache.Total != count {
			t.Errorf("not all collected: %d < %d", cache.Total, count)
		}
	})
	t.Run("Ordering", func(t *testing.T) {
		mock := MockActivityLogSource{}
		waitForSync := 5 * time.Millisecond
		cache := NewJFActivityCache(&mock, waitForSync)
		finished := false
		count := len(cache.cache) * 10
		go mock.run(count, time.Second/100, &finished)
		for {
			if err := cache.MaybeSync(); err != nil {
				t.Errorf("sync failed: %v", err)
				return
			}

			if finished {
				// Make sure we got everything
				time.Sleep(waitForSync)
				if err := cache.MaybeSync(); err != nil {
					t.Errorf("sync failed: %v", err)
					return
				}
				break
			}
		}
		t.Log(">-\n" + cache.debugString())
		i := cache.start
		lastID := int64(-1)
		t.Logf("cache start=%d, end=%d, total=%d\n", cache.start, cache.end, cache.Total)
		for {
			if i != cache.start {
				if cache.cache[i].ID != lastID-1 {
					t.Errorf("next was not previous ID: %d != %d-1 = %d", cache.cache[i].ID, lastID, lastID-1)
					return
				}
			}
			lastID = cache.cache[i].ID

			if i == cache.end {
				break
			}
			i = (i + 1) % len(cache.cache)
		}
	})
}
