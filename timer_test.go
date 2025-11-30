package main

import (
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time                  { return f.now }
func (f fakeClock) Since(t time.Time) time.Duration { return f.now.Sub(t) }

// Tests the timer with negative time deltas, i.e. reminders before an event.
func TestTimerNegative(t *testing.T) {
	as := NewDayTimerSet([]string{
		"1", "2", "3", "7",
	}, -24*time.Hour)

	since := time.Date(2025, 8, 9, 1, 0, 0, 0, time.UTC)
	nowTimes := []time.Time{
		time.Date(2025, 8, 1, 23, 59, 0, 0, time.UTC),
		time.Date(2025, 8, 2, 1, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 3, 7, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 6, 7, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 6, 12, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 7, 5, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 8, 0, 30, 0, 0, time.UTC),
		time.Date(2025, 8, 8, 4, 30, 0, 0, time.UTC),
		time.Date(2025, 8, 8, 16, 30, 0, 0, time.UTC),
	}

	returnValues := []time.Duration{
		0, 7, 0, 3, 0, 2, 1, 0, 0,
	}
	for i := range returnValues {
		returnValues[i] *= -24 * time.Hour
	}

	lastFired := time.Time{}
	for i, nt := range nowTimes {
		target := returnValues[i]

		as.clock = fakeClock{now: nt}

		ret := as.Check(since, lastFired)

		if ret != target {
			t.Fatalf("incorrect return value (%v != %v): i=%d, now=%+v, since=%+v, lastFired=%+v", ret, target, i, nt, since, lastFired)
		}

		if ret != 0 {
			lastFired = nt
		}
	}
}

func TestTimerSmallInterval(t *testing.T) {
	as := NewDayTimerSet([]string{
		"1", "1.1", "2", "3", "7",
	}, -24*time.Hour)

	since := time.Date(2025, 8, 9, 1, 0, 0, 0, time.UTC)
	nowTimes := []time.Time{
		time.Date(2025, 8, 1, 23, 59, 0, 0, time.UTC),
		time.Date(2025, 8, 2, 1, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 3, 7, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 6, 7, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 6, 7, 30, 0, 0, time.UTC),
		time.Date(2025, 8, 6, 12, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 7, 5, 0, 0, 0, time.UTC),
		time.Date(2025, 8, 8, 0, 30, 0, 0, time.UTC),
		time.Date(2025, 8, 8, 4, 30, 0, 0, time.UTC),
		time.Date(2025, 8, 8, 16, 30, 0, 0, time.UTC),
	}

	returnValues := []time.Duration{
		0, 7, 0, 3, 0, 0, 2, 1, 0, 0,
	}
	for i := range returnValues {
		returnValues[i] *= -24 * time.Hour
	}

	lastFired := time.Time{}
	for i, nt := range nowTimes {
		target := returnValues[i]

		as.clock = fakeClock{now: nt}

		ret := as.Check(since, lastFired)

		if ret != target {
			t.Fatalf("incorrect return value (%v != %v): i=%d, now=%+v, since=%+v, lastFired=%+v", ret, target, i, nt, since, lastFired)
		}

		if ret != 0 {
			lastFired = nt
		}
	}
}

func TestTimerBruteForce(t *testing.T) {
	as := NewDayTimerSet([]string{
		"1", "2", "3", "7",
	}, -24*time.Hour)

	since := time.Date(2025, 8, 9, 1, 0, 0, 0, time.UTC)

	returnedValues := map[time.Duration]time.Time{}

	lastFired := time.Time{}
	for dd := range 12 {
		for hh := range 24 {
			for mm := range 60 {
				nt := time.Date(2025, 8, dd, hh, mm, 0, 0, time.UTC)

				as.clock = fakeClock{now: nt}

				ret := as.Check(since, lastFired)

				if dupe, ok := returnedValues[ret]; ok {

					t.Fatalf("duplicate return value (%v): now=%+v, dupe=%+v, since=%+v, lastFired=%+v", ret, nt, dupe, since, lastFired)
				}

				if ret != 0 {
					returnedValues[ret] = nt
					lastFired = nt
				}
			}
		}
	}

	if len(returnedValues) != len(as.deltas) {
		t.Fatalf("not all timers fired (%d/%d)", len(returnedValues), len(as.deltas))
	}
}
