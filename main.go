package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func onMessageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	// ignore own messages
	if message.Author.ID == session.State.User.ID {
		return
	}

	if message.Content == "Hello There" {
		_, err := session.ChannelMessageSend(message.ChannelID, "General Statnobi")
		if err != nil {
			return
		}
	}

	fmt.Printf("Message found: %s by %s\n", message.Content, message.Author.Username)
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
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Println("Failed to create Discord session,", err)
		return
	}

	// register event handlers
	dg.AddHandler(onMessageCreate)

	// open web socket connection
	err = dg.Open()
	if err != nil {
		log.Println("Failed to open the connection,", err)
		return
	}

	fmt.Println("Bot is running...")
	select {}
}
