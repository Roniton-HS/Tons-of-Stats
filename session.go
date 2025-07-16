package main

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

type Session struct {
	dcs *discordgo.Session

	AppID    string
	ServerID string
	Handlers map[string]func()
	Commands map[string]Handler
}

func NewSession(token string, sID string) *Session {
	dcs, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Failed to create session", "sID", sID, "err", err)
	}

	return &Session{dcs, "", sID, make(map[string]func()), make(map[string]Handler)}
}

func (s *Session) Open(cmds []Command) error {
	if err := s.awaitReady(); err != nil {
		return err
	}

	// Register generic handler for all slash-commands.
	s.HandlerAdd("handle-command", func(dcs *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := s.Commands[i.ApplicationCommandData().Name]; ok {
			log.Info("Executing command", "name", i.ApplicationCommandData().Name)

			s.dcs.InteractionRespond(i.Interaction, h(dcs, i.Interaction))
		}
	})

	// Unregister old commands.
	regCmds, err := s.dcs.ApplicationCommands(s.AppID, s.ServerID)
	if err != nil {
		return err
	}

	// PERF: only remove commands not in `cmds`
	for _, c := range regCmds {
		log.Debug("Unregistering left-over command", "id", c.ID, "name", c.Name)
		s.dcs.ApplicationCommandDelete(s.AppID, s.ServerID, c.ID)
	}

	// Register commands.
	for _, c := range cmds {
		if err := s.CommandAdd(c); err != nil {
			return err
		}
	}

	return nil
}

func (s *Session) awaitReady() error {
	var rdy sync.WaitGroup
	rdy.Add(1)

	// Register handler to await session initialization. This ensures tha AppID is
	// available.
	s.HandlerAdd("session-ready", func(dcs *discordgo.Session, r *discordgo.Ready) {
		s.AppID = dcs.State.User.ID
		log.Info("Session ready", "id", s.AppID)
		rdy.Done()
	})

	if err := s.dcs.Open(); err != nil {
		return err
	}

	log.Info("Awaiting session ready")
	rdy.Wait()
	s.HandlerRemove("session-ready")

	return nil
}

func (s *Session) GetUserName(id string) (string, error) {
	member, err := s.dcs.GuildMember(s.ServerID, id)
	if err != nil {
		log.Warn("Failed to get user name", "id", id, "err", err)
		return "", err
	}

	return member.Nick, nil
}

func (s *Session) GetChannelID(name string) (string, error) {
	channels, err := s.dcs.GuildChannels(s.ServerID)
	if err != nil {
		log.Warn("Failed to get channel ID", "name", name, "err", err)
		return "", err
	}

	for _, ch := range channels {
		if ch.Name == name {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("invalid channel name `%s`", name)
}

func (s *Session) MsgSend(chID string, content string) error {
	if _, err := s.dcs.ChannelMessageSend(chID, content); err != nil {
		log.Warn("Failed to send message", "chID", chID, "err", err)
		return err
	}

	log.Info("Message sent", "chID", chID, "content", content)
	return nil
}

func (s *Session) MsgReact(chID string, msgID string, reaction string) error {
	return s.dcs.MessageReactionAdd(chID, msgID, reaction)
}

func (s *Session) CommandAdd(cmd Command) error {
	if _, ok := s.Handlers[cmd.Definition.Name]; ok {
		return fmt.Errorf("command with name `%s` already exists", cmd.Definition.Name)
	}

	if _, err := s.dcs.ApplicationCommandCreate(s.AppID, s.ServerID, cmd.Definition); err != nil {
		return fmt.Errorf("command creation `%s` failed: %v", cmd.Definition.Name, err)
	}

	log.Debug("Command registered", "name", cmd.Definition.Name)
	s.Commands[cmd.Definition.Name] = cmd.Handler
	return nil
}

// Adds an event handler and associates it with the given name. Names must be
// unique to allow deleting them at a later point in time. Errors if a handler
// for the given name already exists.
func (s *Session) HandlerAdd(name string, handler any) error {
	if _, ok := s.Handlers[name]; ok {
		return fmt.Errorf("handler for name `%s` already exists", name)
	}

	log.Debug("Handler registered", "name", name)
	s.Handlers[name] = s.dcs.AddHandler(handler)
	return nil
}

// Removes the event handler for the given name. Results in a noop if no handler
// exists for the name.
func (s *Session) HandlerRemove(name string) {
	if h, ok := s.Handlers[name]; ok {
		log.Debug("Handler removed", "name", name)
		h()
	}
}
