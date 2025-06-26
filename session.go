package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// TODO: don't hardcode this
const ServerID = "1387198610935906305"

type Session struct {
	*discordgo.Session

	ServerID string
}

func NewSession(id string) *Session {
	session, err := discordgo.New("Bot " + id)
	if err != nil {
		log.Fatal("Failed to create session", err)
	}

	return &Session{session, id}
}

func (s Session) GetChannelID(name string) (string, error) {
	channels, err := s.GuildChannels(s.ServerID)
	if err != nil {
		return "", err
	}

	for _, ch := range channels {
		if ch.Name == name {
			return ch.ID, nil
		}
	}

	return "", nil
}

func (s Session) SendMessage(chID string, content string) error {
	_, err := s.ChannelMessageSend(chID, content)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		return err
	}

	log.Printf("Message sent to channel %s", chID)
	return nil
}
