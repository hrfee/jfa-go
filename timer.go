package main

import (
	"strconv"
	"time"
)

const (
	// The maximum duration distance from a trigger time that it will be triggered. If multiple trigger times are provided closer than this value, the smallest will be used instead.
	MAX_MIN_INTERVAL = 18 * time.Hour
)

type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
}

type realClock struct{}

func (realClock) Now() time.Time                  { return time.Now() }
func (realClock) Since(t time.Time) time.Duration { return time.Since(t) }

// DayTimerSet holds information required to trigger timers. Can be generated with NewTimerSet. Does not have it's own event loop, one should check as regularly as they like whether and which of the timers should go off with Check(), passing the last known time one fired (you should track this yourself, the DayTimerSet struct isn't something that needs to be stored).
type DayTimerSet struct {
	deltas []time.Duration
	clock  Clock
}

func NewDayTimerSet(deltaStrings []string, unit time.Duration) *DayTimerSet {
	as := DayTimerSet{
		deltas: make([]time.Duration, 0, len(deltaStrings)),
		clock:  realClock{},
	}
	for i := range deltaStrings {
		// d, err := strconv.ParseFloat(deltaStrings[i], 64)
		d, err := strconv.ParseInt(deltaStrings[i], 10, 64)
		if err == nil {
			as.deltas = append(as.deltas, time.Duration(d)*unit)
		}
	}

	return &as
}

// Returns one or no time.Duration values, Giving the delta for the timer which went off. Pass a non-zero lastFired to stop too many going off at once, and store the returned time.Time value to pass as this later.
func (as DayTimerSet) Check(since time.Time, lastFired time.Time) time.Duration {
	// Keep track of the timer that's most recently gone off, so we don't for example send a "your account expires in 3 days" 1 day away from expiry if the server's been turned off for a while.
	soonestTimerDesiredDelta := time.Duration(0)
	soonestTimerRealDelta := 1e5 * time.Hour
	for _, dt := range as.deltas {
		if dt == time.Duration(0) {
			// fmt.Printf("not firing: zero delta\n")
			continue
		}

		now := as.clock.Now()
		y1, m1, d1 := now.Date()

		if !lastFired.IsZero() {
			y2, m2, d2 := lastFired.Date()
			if y2 == y1 && m2 == m1 && d2 == d1 {
				// fmt.Printf("not firing: same day as last fire (%d.%d.%d == %d.%d.%d)\n", y2, m2, d2, y1, m1, d1)
				continue
			}

			if as.clock.Since(lastFired) < MAX_MIN_INTERVAL {
				// fmt.Printf("not firing: not enough time since last fire (%v < %v)\n", as.clock.Since(lastFired), MAX_MIN_INTERVAL)
				continue
			}
		}

		nd := since.Add(dt)

		y2, m2, d2 := nd.Date()
		if y2 != y1 || m2 != m1 || d2 != d1 {
			// fmt.Printf("not firing: not same day (%d.%d.%d != %d.%d.%d)\n", y2, m2, d2, y1, m1, d1)
			continue
		}
		dNowNotif := now.Sub(nd).Abs()

		if dNowNotif > MAX_MIN_INTERVAL {
			// fmt.Printf("not firing: not close enough to fire time (%v > %v)\n", dNowNotif, MAX_MIN_INTERVAL)
			continue
		}

		if dNowNotif < soonestTimerRealDelta {
			soonestTimerDesiredDelta = dt
			soonestTimerRealDelta = dNowNotif
		}
	}
	return soonestTimerDesiredDelta
}
