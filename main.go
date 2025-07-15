package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

type StatsToday struct {
	UserID string

	Classic      int
	Quote        int
	Ability      int
	AbilityCheck bool
	Emoji        int
	Splash       int
	SplashCheck  bool

	EloChange float64
}

func (s *StatsToday) String() string {
	name, err := session.GetUserName(s.UserID)
	if err != nil {
		return "Something went wrong :\\"
	}

	aChk, sChk := "", ""
	if s.AbilityCheck {
		aChk = "✔"
	}
	if s.SplashCheck {
		sChk = "✔"
	}

	return fmt.Sprintf(
		`%s
%s
Classic %d
Quote   %d
Ability %d %s
Emoji   %d
Splash  %d %s
`,
		fmt.Sprintf("\x1b\n%s", name),
		strings.Repeat("─", utf8.RuneCountInString(name)),
		s.Classic,
		s.Quote,
		s.Ability,
		aChk,
		s.Emoji,
		s.Splash,
		sChk,
	)
}

type StatsTotal struct {
	UserID string

	Classic      int
	Quote        int
	Ability      int
	AbilityCheck int
	Emoji        int
	Splash       int
	SplashCheck  int

	DaysPlayed int
	Elo        float64
}

var db *StatsDB
var session *Session

// Schedules a job to be run repeatedly with the given start time and interval.
// The job is always run at least once, after the start time has elapsed. If the
// start time is in the past, the first invocation occurs immediately. If
// interval is 0, the function exits after the first invocation.
func schedule(start time.Time, interval time.Duration, job func()) {
	delay := time.Until(start)
	log.Info("Scheduling job", "job", job, "delay", delay.Round(time.Second), "interval", interval)

	timer := time.NewTimer(delay)
	<-timer.C

	// first job invocation
	log.Info("Running job", "job", job)
	go func() {
		job()
		log.Info("Job complete", "job", job)
	}()

	if interval == 0 {
		return
	}

	// repeat
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

// Work to be performed once every day at midnight.
//
// Responsible for printing daily stats for all users, the current global
// leaderboard standings, as well as resetting the recorded daily stats.
func dailyReset() {
	chID, err := session.GetChannelID("daily-stats")
	if err != nil {
		log.Warn("Invalid channel", "name", "daily-stats")
		return
	}

	entries, err := db.Today.GetAll()
	if err != nil {
		log.Error("Failed to fetch", "table", "today", "err", err)
	}

	if len(entries) > 0 {
		session.MsgSend(chID, "Today's Stats:")
	}
	for _, entry := range entries {
		session.MsgSend(chID, entry.String())
	}

	if err := db.Today.DeleteAll(); err != nil {
		log.Error("Failed to delete from table", "table", "today", "err", err)
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
		log.Warn("No .env file present", "err", err)
	}

	token, ok := os.LookupEnv("DISCORD_BOT_TOKEN")
	if !ok {
		log.Fatal("DISCORD_BOT_TOKEN not set")
	}
	server, ok := os.LookupEnv("SERVER_ID")
	if !ok {
		log.Fatal("SERVER_ID not set")
	}

	// Database configuration
	conn, err := sql.Open("sqlite3", "tons_of_stats.sqlite")
	if err != nil {
		log.Fatal("Failed to open database", "err", err)
	}

	db = NewStatsDB(conn)
	if err := db.Setup(); err != nil {
		log.Fatal("Failed to set up database", "err", err)
	}
	defer db.Close()

	// Discord session configuration
	session = NewSession(token, server)
	if err := session.Open(); err != nil {
		log.Fatal("Failed to open session", "err", err)
	}

	session.HandlerAdd("record-stats", recordStats)
	session.HandlerAdd("display-stats", displayStats)

	// Automated message scheduling
	now := time.Now()
	midnight := time.Date(
		now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location(),
	)
	go schedule(midnight, 24*time.Hour, dailyReset)

	log.Info("Running...")
	select {}
}
