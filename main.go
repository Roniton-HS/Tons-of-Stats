package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var userStatMap = make(map[string]map[string]float64)

// Listens to messages sent in the result-spam channel
func onMessageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if strings.HasPrefix(message.Content, "stats") {
		sendDailyStats(session, message.ChannelID)
		return
	}
	if message.ChannelID != getChannelIDByName(session, message.GuildID, "result-spam") {
		return
	}
	if !strings.HasPrefix(message.Content, "I've completed all the modes of #LoLdle today:") {
		return
	}

	lines := strings.Split(message.Content, "\n")
	stats := make(map[string]float64)
	for i, line := range lines {
		if i == 0 {
			continue
		}

		// remove emoji
		lineParts := strings.SplitN(line, " ", 2)
		if len(lineParts) < 2 {
			continue
		}

		remainingLine := lineParts[1]

		// split key and value
		dataParts := strings.SplitN(remainingLine, ":", 2)
		if len(dataParts) < 2 {
			continue
		}

		key := strings.TrimSpace(dataParts[0])
		value := strings.TrimSpace(dataParts[1])

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

	userStatMap[message.Author.ID] = stats
}

// Schedules tasks to run at midnight
func scheduleMidnight(session *discordgo.Session, channelID string) {
	go func() {
		for {
			now := time.Now()
			nextMidnight := time.Date(
				now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location(),
			)
			durationUntilNextMidnight := time.Until(nextMidnight)
			time.Sleep(durationUntilNextMidnight)

			// Do this at midnight
			// Todo: generate a message that displays the "elo" of all tracked users
			sendDailyStats(session, channelID)
		}
	}()
}

func sendDailyStats(session *discordgo.Session, channelID string) {
	// Todo: make this nice

	var output = ""
	for userID, stats := range userStatMap {
		user, err := session.GuildMember("1387198610935906305", userID)
		if err != nil {
			log.Printf("Failed get username: %v", err)
		}

		output += user.DisplayName() + ":\n"
		for key, value := range stats {
			output += fmt.Sprintf("%s: %g\n", key, value)
		}
		output += "\n"
	}
	sendMessage(session, channelID, output)
}

func sendMessage(session *discordgo.Session, channelID string, content string) {
	_, err := session.ChannelMessageSend(channelID, content)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	} else {
		log.Printf("Message sent to channel %s", channelID)
	}
}

func getChannelIDByName(session *discordgo.Session, serverID string, channelName string) string {
	channels, err := session.GuildChannels(serverID)
	if err != nil {
		log.Println("Failed to get channelID", err)
		return ""
	}

	for _, channel := range channels {
		if channel.Name == channelName {
			return channel.ID
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
		log.Println("Failed to create Discord session,", err)
		return
	}

	// register event handlers
	session.AddHandler(onMessageCreate)

	// open web socket connection
	err = session.Open()
	if err != nil {
		log.Println("Failed to open the connection,", err)
		return
	}

	// hardcoded serverID for now
	scheduleMidnight(session, getChannelIDByName(session, "1387198610935906305", "result-spam"))

	fmt.Println("Bot is running...")
	select {}
}
