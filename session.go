package main

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

type Session struct {
	dcs *discordgo.Session

	ServerID string
}

func NewSession(token string, sID string) *Session {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Failed to create session", "sID", sID, "err", err)
	}

	return &Session{session, sID}
}

func (s Session) Open() error {
	var rdy sync.WaitGroup
	rdy.Add(1)

	session.dcs.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Info("Session ready", "id", s.State.User.ID)
		rdy.Done()
	})

	if err := s.dcs.Open(); err != nil {
		return err
	}

	log.Info("Awaiting session ready")
	rdy.Wait()
	return nil
}

func (s Session) GetUserName(id string) (string, error) {
	member, err := s.dcs.GuildMember(s.ServerID, id)
	if err != nil {
		return "", err
	}

	return member.Nick, nil
}

func (s Session) GetChannelID(name string) (string, error) {
	channels, err := s.dcs.GuildChannels(s.ServerID)
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

func (s Session) MsgSend(chID string, content string) error {
	_, err := s.dcs.ChannelMessageSend(chID, content)
	if err != nil {
		log.Warn("Failed to send message", "chID", chID, "err", err)
		return err
	}

	log.Info("Message sent", "chID", chID, "content", content)
	return nil
}

func (s Session) MsgReact(chID string, msgID string, reaction string) error {
	return s.dcs.MessageReactionAdd(chID, msgID, reaction)
}
