package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
	"time"
)

// Listens to messages sent in the result-spam channel
func onMessageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.ChannelID != getChannelIDByName(session, message.GuildID, "result-spam") {
		return
	}

	if !strings.HasPrefix(message.Content, "I've completed all the modes of #LoLdle today:") {
		return
	}

	// Todo: save stats
	sendMessage(session, message.ChannelID, "LOL, you suck!")

	log.Printf("Message found: %s by %s\n", message.Content, message.Author.Username)
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
			// Todo: generate a message that compares the daily results of all tracked users
			sendMessage(session, channelID, "It's midnight my dudes!")
		}
	}()
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
