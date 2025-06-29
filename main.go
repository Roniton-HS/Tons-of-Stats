package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

type StatsToday struct {
	*CmpStats

	UserID    string
	EloChange float64
}

var db *DB
var session *Session

// Handle requests for stat-display.
//
// TODO: refactor to proper command
func handleStatDisplay(_ *discordgo.Session, msg *discordgo.MessageCreate) {
	if strings.TrimSpace(msg.Content) != "stats" {
		return
	}

	stats, err := db.GetStatsToday(msg.Author.ID)
	if err != nil {
		log.Warn("Stat retrieval failed", "chID", msg.ChannelID, "uID", msg.Author.ID, "err", err)
		session.SendMessage(msg.ChannelID, "Sowwy, I could not retwieve your stats owO")
		return
	}

	sb := strings.Builder{}
	sb.Write(fmt.Appendf(nil, "Stats fow %s UwU:\n", msg.Member.Nick))
	sb.Write([]byte(stats.String()))
	sb.Write([]byte("\n"))

	session.SendMessage(msg.ChannelID, sb.String())
}

// Handle LoLdle result messages and update user stats accordingly.
func handleUserStats(_ *discordgo.Session, msg *discordgo.MessageCreate) {
	if ch, err := session.GetChannelID("result-spam"); err != nil || msg.ChannelID != ch {
		return
	} else if !strings.HasPrefix(msg.Content, "I've completed all the modes of #LoLdle today:") {
		return
	}

	stats, err := ParseStats(msg.Content)
	if err != nil {
		log.Error("Message parsing failed", "err", err)
		// TODO: error message
		return
	}

	// TODO: elo calculation
	if err := db.SetStatsToday(&StatsToday{stats, msg.Author.ID, 0}); err != nil {
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

	// open and configure database
	db = NewDB()
	if err := db.Setup(); err != nil {
		log.Fatal("Failed to set up database", "err", err)
	}

	// create and initialize new session
	session = NewSession(token, ServerID)
	session.AddHandler(handleStatDisplay)
	session.AddHandler(handleUserStats)

	if err := session.Open(); err != nil {
		log.Fatal("Failed to open connection", "err", err)
	}

	// schedule stat-messages
	go scheduleMidnight(func() {
		_, err := session.GetChannelID("result-spam")
		if err != nil {
			log.Warn("Invalid channel", "name", "result-spam")
			return
		}

		// TODO: stat display for all users
	})

	log.Info("Running...")
	select {}
}
