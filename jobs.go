package main

import (
	"time"

	"github.com/charmbracelet/log"
)

// schedule registers a job (i.e. callback) to be run repeatedly. The first
// invocation happens at start, from where on out the job is invoked at the
// given interval.
//
// The job is always run at least once, when the start time elapses. If the
// start time is in the past, the first invocation occurs immediately. If
// interval is 0, the function exits after this first invocation.
func schedule(start time.Time, interval time.Duration, job func()) {
	delay := time.Until(start)
	log.Info("Scheduling job", "job", job, "delay", delay.Round(time.Second), "interval", interval)

	timer := time.NewTimer(delay)
	<-timer.C

	// First job invocation after initial delay.
	log.Info("Running job", "job", job)
	go func() {
		job()
		log.Debug("Job complete", "job", job)
	}()

	if interval == 0 {
		return
	}

	// Repeat invocations on every tick.
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		log.Info("Running job", "job", job)
		go func() {
			job()
			log.Debug("Job complete", "job", job)
		}()
	}
}

// dailyReset specifies the work to be performed by the bot once a day at
// midnight.
func dailyReset() {
	log.Info("Performing daily reset")

	// Delete all entries from the daily stats table. This is necessary, since we
	// use primary key conflicts in the database layer to detect repeat
	// submissions within the same day. Using a separate data structure does not
	// offer persistance across application restarts.
	if err := dal.Today.DeleteAll(); err != nil {
		log.Error("Failed to clear daily stats", "table", dal.Today.Tbl, "err", err)
	}

	leaderboard.Update()
}
