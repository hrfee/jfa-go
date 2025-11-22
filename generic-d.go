package main

import (
	"time"

	lm "github.com/hrfee/jfa-go/logmessages"
)

// https://bbengfort.github.io/snippets/2016/06/26/background-work-goroutines-timer.html THANKS

type GenericDaemon struct {
	Stopped         bool
	ShutdownChannel chan string
	TriggerChannel  chan bool
	Interval        time.Duration
	period          time.Duration
	jobs            []func(app *appContext)
	app             *appContext
	name            string
}

func (d *GenericDaemon) appendJobs(jobs ...func(app *appContext)) {
	d.jobs = append(d.jobs, jobs...)
}

// NewGenericDaemon returns a daemon which can be given jobs that utilize appContext.
func NewGenericDaemon(interval time.Duration, app *appContext, jobs ...func(app *appContext)) *GenericDaemon {
	d := GenericDaemon{
		Stopped:         false,
		ShutdownChannel: make(chan string),
		TriggerChannel:  make(chan bool),
		Interval:        interval,
		period:          interval,
		app:             app,
		name:            "Generic Daemon",
	}
	d.jobs = jobs
	return &d

}

func (d *GenericDaemon) Name(name string) { d.name = name }

func (d *GenericDaemon) run() {
	d.app.info.Printf(lm.StartDaemon, d.name)
	for {
		select {
		case <-d.ShutdownChannel:
			d.ShutdownChannel <- "Down"
			return
		case <-d.TriggerChannel:
			break
		case <-time.After(d.period):
			break
		}
		started := time.Now()

		for _, job := range d.jobs {
			job(d.app)
		}

		finished := time.Now()
		duration := finished.Sub(started)
		d.period = d.Interval - duration
	}
}

func (d *GenericDaemon) Trigger() {
	d.TriggerChannel <- true
}

func (d *GenericDaemon) Shutdown() {
	d.Stopped = true
	d.ShutdownChannel <- "Down"
	<-d.ShutdownChannel
	close(d.ShutdownChannel)
}
