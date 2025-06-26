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

// Listens to messages sent in the result-spam channel
func onMessageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if strings.HasPrefix(message.Content, "stats") {
		sendDailyStats(session, message.ChannelID)
		return
	} else if message.ChannelID != getChannelIDByName(session, message.GuildID, "result-spam") {
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

// Schedules tasks to run at midnight
func scheduleMidnight(session *discordgo.Session, channelID string) {
	for {
		now := time.Now()
		nextMidnight := time.Date(
			now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location(),
		)
		durationUntilNextMidnight := time.Until(nextMidnight)
		time.Sleep(durationUntilNextMidnight)

		// Do this at midnight
		// TODO: generate a message that displays the "elo" of all tracked users
		sendDailyStats(session, channelID)
	}
}

func sendDailyStats(session *discordgo.Session, channelID string) {
	// TODO: make this nice

	str := ""
	for userID, stats := range userStats {
		user, err := session.GuildMember("1387198610935906305", userID)
		if err != nil {
			log.Printf("Failed get username: %v", err)
		}

		str += user.DisplayName() + ":\n"
		for key, value := range stats {
			str += fmt.Sprintf("%s: %g\n", key, value)
		}
		str += "\n"
	}
	sendMessage(session, channelID, str)
}

func sendMessage(session *discordgo.Session, channelID string, content string) {
	_, err := session.ChannelMessageSend(channelID, content)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		return
	}

	log.Printf("Message sent to channel %s", channelID)
}

func getChannelIDByName(session *discordgo.Session, serverID string, channelName string) string {
	channels, err := session.GuildChannels(serverID)
	if err != nil {
		log.Println("Failed to get channelID", err)
		return ""
	}

	for _, ch := range channels {
		if ch.Name == channelName {
			return ch.ID
		}
	}

	log.Println("No channel found with name: ", channelName)
	return ""
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load .env file", err)
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN not found in .env")
	}

	// create a new session
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Failed to create Discord session", err)
	}

	// register event handlers
	session.AddHandler(onMessageCreate)

	// open web socket connection
	err = session.Open()
	if err != nil {
		log.Fatal("Failed to open connection", err)
	}

	// TODO: don't hardcode server ID
	go scheduleMidnight(session, getChannelIDByName(session, "1387198610935906305", "result-spam"))

	fmt.Println("Bot is running...")
	select {}
}
