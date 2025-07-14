package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

type Session struct {
	*discordgo.Session

	ServerID string
}

func NewSession(token string, sID string) *Session {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Failed to create session", "sID", sID, "err", err)
	}

	return &Session{session, sID}
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

	return "", fmt.Errorf("invalid channel name `%s`", name)
}

func (s Session) SendMessage(chID string, content string) error {
	_, err := s.ChannelMessageSend(chID, content)
	if err != nil {
		log.Warn("Failed to send message", "chID", chID, "err", err)
		return err
	}

	log.Info("Message sent", "chID", chID, "content", content)
	return nil
}
