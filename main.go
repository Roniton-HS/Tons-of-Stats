package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

type StatsToday struct {
	*LoldleStats

	UserID    string
	EloChange float64
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
		log.Warn("Stat retrieval failed", "chID", msg.ChannelID, "uID", msg.Author.ID, "err", err)
		session.SendMessage(msg.ChannelID, "Failed to retrieve your stats.")
		return
	}

	sb := strings.Builder{}
	sb.Write(fmt.Appendf(nil, "%s's stats for today:\n", msg.Member.Nick))
	sb.Write([]byte(stats.String()))
	sb.Write([]byte("\n"))

	session.SendMessage(msg.ChannelID, sb.String())
}

// Handle LoLdle result messages and update user stats accordingly.
func handleUserStats(_ *discordgo.Session, msg *discordgo.MessageCreate) {
	if ch, err := session.GetChannelID("result-spam"); err != nil || msg.ChannelID != ch {
		return
	} else if !strings.HasPrefix(msg.Content, LoldleHeader) {
		return
	}

	stats, err := ParseStats(msg.Content)
	if err != nil {
		log.Error("Message parsing failed", "err", err)
		// TODO: error message
		return
	}

	// TODO:
	// + elo calculation
	// + update total stats
	if err := db.Today.Update(msg.Author.ID, &StatsToday{stats, msg.Author.ID, 0}); err != nil {
		log.Warn("Failed to record daily stats", "user", msg.Author.ID, "msg", msg.Content, "err", err)
		session.MessageReactionAdd(msg.ChannelID, msg.ID, "❌")
	} else {
		log.Info("Daily stats recorded", "user", msg.Author.ID)
		session.MessageReactionAdd(msg.ChannelID, msg.ID, "✅")
	}
}

// Schedules job to run daily at midnight
func scheduleMidnight(job func()) {
	now := time.Now()
	midnight := time.Date(
		now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location(),
	)

	timer := time.NewTimer(time.Until(midnight))
	<-timer.C

	// first job invocation at midnight
	go job()

	// repeat every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	for {
		<-ticker.C
		go job()
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
	session = NewSession(token, ServerID)
	session.AddHandler(handleStatDisplay)
	session.AddHandler(handleUserStats)

	if err := session.Open(); err != nil {
		log.Fatal("Failed to open session", "err", err)
	}

	// Automated message scheduling
	go scheduleMidnight(func() {
		_, err := session.GetChannelID("result-spam")
		if err != nil {
			log.Warn("Invalid channel", "name", "result-spam")
			return
		}

		// TODO: stat display for all users, drop `today` table
	})

	log.Info("Running...")
	select {}
}
