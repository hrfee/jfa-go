package main

import "time"

// https://bbengfort.github.io/snippets/2016/06/26/background-work-goroutines-timer.html THANKS

type Repeater struct {
	Stopped         bool
	ShutdownChannel chan string
	Interval        time.Duration
	period          time.Duration
	app             *appContext
}

func NewRepeater(interval time.Duration, app *appContext) *Repeater {
	return &Repeater{
		Stopped:         false,
		ShutdownChannel: make(chan string),
		Interval:        interval,
		period:          interval,
		app:             app,
	}
}

func (rt *Repeater) Run() {
	rt.app.info.Println("Invite daemon started")
	for {
		select {
		case <-rt.ShutdownChannel:
			rt.ShutdownChannel <- "Down"
			return
		case <-time.After(rt.period):
			break
		}
		started := time.Now()
		rt.app.storage.loadInvites()
		rt.app.debug.Println("Daemon: Checking invites")
		rt.app.checkInvites()
		finished := time.Now()
		duration := finished.Sub(started)
		rt.period = rt.Interval - duration
	}
}

func (rt *Repeater) Shutdown() {
	rt.Stopped = true
	rt.ShutdownChannel <- "Down"
	<-rt.ShutdownChannel
	close(rt.ShutdownChannel)
}
