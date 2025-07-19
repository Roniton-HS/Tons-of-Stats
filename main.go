package main

import (
	"time"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

var dal *DAL
var env *Env
var session *Session

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
		log.Info("Job complete", "job", job)
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
			log.Info("Job complete", "job", job)
		}()
	}
}

// dailyReset specifies the work to be performed by the bot once a day at
// midnight.
//
// The job is responsible for printing daily stats for all users, the current
// global leaderboard standings, as well as resetting the recorded daily stats.
func dailyReset() {
	log.Info("Performing daily reset")
	chID, err := session.GetChannelID(env.StatsCh)
	if err != nil {
		log.Warn("Invalid channel", "name", env.StatsCh)
		return
	}

	// Get all stats recorded today and create messages for each.
	entries, err := dal.Today.GetAll()
	if err != nil {
		log.Error("Failed to fetch", "table", "today", "err", err)
	}

	if len(entries) > 0 {
		session.MsgSend(chID, "Today's Stats:")
	}
	for _, entry := range entries {
		session.MsgSend(chID, entry.String())
	}

	// TODO: leaderboard update

	// Delete all entries from the daily stats table. This is necessary, since we
	// use primary key conflicts in the database layer to detect repeat
	// submissions within the same day. Using a separate data structure does not
	// offer persistance across application restarts.
	if err := dal.Today.DeleteAll(); err != nil {
		log.Error("Failed to clear daily stats", "table", dal.Today.Tbl, "err", err)
	}
}

func main() {
	log.SetDefault(
		log.NewWithOptions(nil, log.Options{
			ReportCaller:    true,
			ReportTimestamp: true,
			TimeFormat:      time.Kitchen,
			Level:           log.DebugLevel,
		}),
	)

	err := godotenv.Load()
	if err != nil {
		log.Warn("Failed to load .env", "err", err)
	}

	env = NewEnv()
	if env.IsProd {
		log.SetLevel(log.InfoLevel)
	}

	// Database configuration
	db, err := NewDB("tons_of_stats.sqlite")
	if err != nil {
		log.Fatal("Could not open database", "err", err)
	}
	defer db.Close()

	dal = NewDAL(db)

	// Discord session configuration
	session = NewSession(env.Token, env.ServerID)
	if err := session.Open(cmds); err != nil {
		log.Fatal("Failed to open session", "err", err)
	}

	session.HandlerAdd("record-stats", RecordStats)

	// Automated message scheduling
	now := time.Now()
	midnight := time.Date(
		now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location(),
	)
	go schedule(midnight, 24*time.Hour, dailyReset)

	log.Info("Running...")
	select {}
}
