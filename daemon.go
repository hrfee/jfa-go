package main

import "time"

// https://bbengfort.github.io/snippets/2016/06/26/background-work-goroutines-timer.html THANKS

type Repeater struct {
	Stopped         bool
	ShutdownChannel chan string
	Interval        time.Duration
	period          time.Duration
	ctx             *appContext
}

func NewRepeater(interval time.Duration, ctx *appContext) *Repeater {
	return &Repeater{
		Stopped:         false,
		ShutdownChannel: make(chan string),
		Interval:        interval,
		period:          interval,
		ctx:             ctx,
	}
}

func (rt *Repeater) Run() {
	rt.ctx.info.Println("Invite daemon started")
	for {
		select {
		case <-rt.ShutdownChannel:
			rt.ShutdownChannel <- "Down"
			return
		case <-time.After(rt.period):
			break
		}
		started := time.Now()
		rt.ctx.storage.loadInvites()
		rt.ctx.debug.Println("Daemon: Checking invites")
		rt.ctx.checkInvites()
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
