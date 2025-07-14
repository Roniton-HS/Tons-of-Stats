package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
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

// Handle requests for stat-display.
//
// TODO: refactor to proper command
func handleStatDisplay(_ *discordgo.Session, msg *discordgo.MessageCreate) {
	if strings.TrimSpace(msg.Content) != "stats" {
		return
	}

	stats, err := db.Today.Get(msg.Author.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			session.SendMessage(msg.ChannelID, "No stats recorded today.")
			return
		}

		log.Warn("Stat retrieval failed", "chID", msg.ChannelID, "uID", msg.Author.ID, "err", err)
		session.SendMessage(msg.ChannelID, "Failed to retrieve your stats.")
		return
	}

	session.SendMessage(msg.ChannelID, stats.String())
}

// Handle LoLdle result messages and update user stats accordingly.
func handleUserStats(_ *discordgo.Session, msg *discordgo.MessageCreate) {
	if ch, err := session.GetChannelID("result-spam"); err != nil || msg.ChannelID != ch {
		return
	} else if !strings.HasPrefix(msg.Content, LoldleHeader) {
		// TODO: error message
		return
	}

	lStats, err := ParseStats(msg.Content)
	if err != nil {
		log.Error("Message parsing failed", "err", err)
		// TODO: error message
		return
	}

	// TODO:
	// + elo calculation
	// + update total stats
	sToday := &StatsToday{
		msg.Author.ID,
		lStats.Classic,
		lStats.Quote,
		lStats.Ability,
		lStats.AbilityCheck,
		lStats.Emoji,
		lStats.Splash,
		lStats.SplashCheck,
		0,
	}

	if err := db.Today.Update(msg.Author.ID, sToday); err != nil {
		log.Warn("Failed to record daily stats", "user", msg.Author.ID, "msg", msg.Content, "err", err)
		session.MessageReactionAdd(msg.ChannelID, msg.ID, "❌")
	} else {
		log.Info("Daily stats recorded", "user", msg.Author.ID)
		session.MessageReactionAdd(msg.ChannelID, msg.ID, "✅")
	}
}

// Schedules a job to be run repeatedly with the given start time and interval.
// The job is always run at least once, after the start time has elapsed. If the
// start time is in the past, the first invocation occurs immediately. If
// interval is 0, the function exits after the first invocation.
func schedule(start time.Time, interval time.Duration, job func()) {
	delay := time.Until(start)
	log.Info("Scheduling job", "delay", delay.Round(time.Second), "interval", interval, "job", job)

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
	session.AddHandler(handleStatDisplay)
	session.AddHandler(handleUserStats)

	if err := session.Open(); err != nil {
		log.Fatal("Failed to open session", "err", err)
	}

	// Automated message scheduling
	now := time.Now()
	midnight := time.Date(
		now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location(),
	)
	go schedule(midnight, 24*time.Hour, func() {
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
			session.SendMessage(chID, "Today's Stats:")
		}
		for _, entry := range entries {
			session.SendMessage(chID, entry.String())
		}

		if err := db.Today.DeleteAll(); err != nil {
			log.Error("Failed to delete from table", "table", "today", "err", err)
		}
	})

	log.Info("Running...")
	select {}
}
