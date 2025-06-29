package main

import (
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

type Stats = map[string]float64
type User = string

var userStats = make(map[User]*CmpStats)
var session *Session

// Handle requests for stat-display.
//
// TODO: refactor to proper command
func handleStatDisplay(_ *discordgo.Session, message *discordgo.MessageCreate) {
	if strings.TrimSpace(message.Content) != "stats" {
		return
	}

	sendDailyStats(session, message.ChannelID)
}

// Handle LoLdle result messages and update user stats accordingly.
func handleUserStats(_ *discordgo.Session, message *discordgo.MessageCreate) {
	if ch, err := session.GetChannelID("result-spam"); err != nil || message.ChannelID != ch {
		return
	} else if !strings.HasPrefix(message.Content, "I've completed all the modes of #LoLdle today:") {
		return
	}

	stats, err := ParseStats(message.Content)
	if err != nil {
		log.Error("Message parsing failed", "err", err)
		// TODO: send error message
		return
	}

	userStats[message.Author.ID] = stats
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

// TODO: make this nice
func sendDailyStats(session *Session, chID string) {
	str := ""
	for uID, stats := range userStats {
		user, err := session.GuildMember(session.ServerID, uID)
		if err != nil {
			log.Warn("Failed to get user", "uID", uID, "err", err)
		}

		str += user.DisplayName() + ":\n"
		str += stats.String()
		str += "\n"
	}

	session.SendMessage(chID, str)
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

	// create and initialize new session
	session = NewSession(token, ServerID)
	session.AddHandler(handleStatDisplay)
	session.AddHandler(handleUserStats)

	if err := session.Open(); err != nil {
		log.Fatal("Failed to open connection", "err", err)
	}

	// schedule stat-messages
	go scheduleMidnight(func() {
		chID, err := session.GetChannelID("result-spam")
		if err != nil {
			log.Warn("Invalid channel", "name", "result-spam")
			return
		}

		// TODO: generate a message that displays the "elo" of all tracked users
		sendDailyStats(session, chID)
	})

	log.Info("Running...")
	select {}
}
