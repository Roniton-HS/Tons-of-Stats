package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type Stats = map[string]float64
type User = string

var userStats = make(map[User]Stats)
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

	stats := make(Stats)
	for _, ln := range strings.Split(message.Content, "\n")[1:] {
		// remove emoji
		words := strings.SplitN(ln, " ", 2)
		if len(words) < 2 {
			continue
		}

		// split key and value
		data := strings.SplitN(words[1], ":", 2)
		if len(data) < 2 {
			continue
		}

		key, value := strings.TrimSpace(data[0]), strings.TrimSpace(data[1])

		isChecked := strings.HasSuffix(value, "✓")
		if isChecked {
			value = strings.TrimSuffix(value, "✓")
			value = strings.TrimSpace(value)
		}

		intValue, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("Failed to convert value '%s' to an integer: %v", value, err)
			continue
		}

		floatValue := float64(intValue)
		if isChecked {
			floatValue -= 0.5
		}

		stats[key] = floatValue
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
	for userID, stats := range userStats {
		user, err := session.GuildMember(ServerID, userID)
		if err != nil {
			log.Printf("Failed to get username: %v", err)
		}

		str += user.DisplayName() + ":\n"
		for key, value := range stats {
			str += fmt.Sprintf("%s: %g\n", key, value)
		}
		str += "\n"
	}

	session.SendMessage(chID, str)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("No .env file found: %v", err)
	}

	token, ok := os.LookupEnv("DISCORD_BOT_TOKEN")
	if !ok {
		log.Fatal("DISCORD_BOT_TOKEN not found in .env")
	}

	// create and initialize new session
	session = NewSession(token, ServerID)
	session.AddHandler(handleStatDisplay)
	session.AddHandler(handleUserStats)

	if err := session.Open(); err != nil {
		log.Fatalf("Failed to open connection: %v", err)
	}

	// schedule stat-messages
	go scheduleMidnight(func() {
		chID, err := session.GetChannelID("result-spam")
		if err != nil {
			log.Printf("Invalid channel name: '%s'", "result-spam")
			return
		}

		// TODO: generate a message that displays the "elo" of all tracked users
		sendDailyStats(session, chID)
	})

	log.Println("Bot is running...")
	select {}
}
