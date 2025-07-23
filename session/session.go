package session

import (
	"fmt"
	"reflect"
	"slices"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

// Message flag for sending V2 components. Not yet available in discordgo.
//
// https://discord.com/developers/docs/components/reference#component-reference
const IS_COMPONENTS_V2 = 1 << 15

// Session is a connection to a [*discordgo.Session] with additional metadata as
// well as all registered event handlers (see [discordgo.EventHandler])
// slash-commands (see [discordgo.ApplicationCommand]).
type Session struct {
	// The underlying session.
	dcs *discordgo.Session

	// Application ID associated with the bot.
	AppID string

	// Server ID the session is connected to (see [discordgo.Guild]).
	ServerID string

	// Maps registered event handler names to their cancellation callbacks (see
	// [discordgo.Session.AddHandler]).
	Handlers map[string]func()

	// Maps registered command names to their handler functions.
	Commands map[string]Handler
}

// NewSession creates a new session, connecting the application to the given
// server.
func NewSession(token string, sID string) *Session {
	dcs, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Failed to create session", "sID", sID, "err", err)
	}

	return &Session{dcs, "", sID, make(map[string]func()), make(map[string]Handler)}
}

// Open configures the underlying session.
func (s *Session) Open(cmds []Command) error {
	if err := s.awaitReady(); err != nil {
		return err
	}

	// Register generic handler for all slash-commands.
	s.HandlerAdd("handle-command", func(dcs *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := s.Commands[i.ApplicationCommandData().Name]; ok {
			log.Info("Executing command", "name", i.ApplicationCommandData().Name)
			if err := s.dcs.InteractionRespond(i.Interaction, h(dcs, i.Interaction)); err != nil {
				log.Error("Execution failed", "name", i.ApplicationCommandData().Name, "err", err)
			}
		}
	})

	// Unregister left-over commands. Application commands are registered on the
	// server itself. Deprecations or changes to command names leave behind
	// "ghost"-commands that don't work and simply produce an error.
	appCmds, err := s.dcs.ApplicationCommands(s.AppID, s.ServerID)
	if err != nil {
		return err
	}

	for _, c := range appCmds {
		if slices.ContainsFunc(cmds, func(c2 Command) bool { return c.Name == c2.Definition.Name }) {
			continue
		}

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

// awaitReady starts initialization of the underlying session and synchronously
// waits for the initialization to finish.
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

// GetUserName returns the server-local nickname for the user with the given ID.
func (s *Session) GetUserName(id string) (string, error) {
	member, err := s.dcs.GuildMember(s.ServerID, id)
	if err != nil {
		log.Warn("Failed to get user name", "id", id, "err", err)
		return "", err
	}

	return member.Nick, nil
}

// GetChannelID returns the ID for the channel with the given name.
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

// MsgList returns as many messages as can be found from the channel with the
// given ID.
func (s *Session) MsgList(chID string) ([]*discordgo.Message, error) {
	return s.dcs.ChannelMessages(chID, 100, "", "", "")
}

// MsgSend sends a message with contents content to the channel with ID chID.
func (s *Session) MsgSend(chID string, content string) (*discordgo.Message, error) {
	m, err := s.dcs.ChannelMessageSend(chID, content)
	if err != nil {
		log.Warn("Failed to send message", "chID", chID, "err", err)
		return nil, err
	}

	log.Info("Message sent", "chID", chID, "content", content)
	return m, nil
}

// MsgSendComplex sends a Components V2 message to the given channel (see
// [IS_COMPONENTS_V2]).
func (s *Session) MsgSendComplex(chID string, cmp []discordgo.MessageComponent) (*discordgo.Message, error) {
	send := discordgo.MessageSend{
		Flags:      IS_COMPONENTS_V2,
		Components: cmp,
	}

	m, err := s.dcs.ChannelMessageSendComplex(chID, &send)
	if err != nil {
		log.Warn("Failed to send message", "chID", chID, "msg", send, "err", err)
		return nil, err
	}

	log.Info("Message sent", "chID", chID, "msg", send)
	return m, nil
}

// MsgEditComplex edits the Components V2 message with the given ID in the given
// channel (see [IS_COMPONENTS_V2]).
func (s *Session) MsgEditComplex(chID string, msgID string, cmp []discordgo.MessageComponent) (*discordgo.Message, error) {
	edit := discordgo.MessageEdit{
		Channel:    chID,
		ID:         msgID,
		Flags:      IS_COMPONENTS_V2,
		Components: &cmp,
	}

	m, err := s.dcs.ChannelMessageEditComplex(&edit)
	if err != nil {
		log.Warn("Failed to edit message", "chID", chID, "msgID", msgID, "msg", edit, "err", err)
		return nil, err
	}

	log.Info("Message edited", "chID", chID, "msgID", msgID, "msg", edit)
	return m, nil
}

// MsgReact adds a reaction to the given message, in the given channel.
func (s *Session) MsgReact(chID string, msgID string, reaction string) error {
	return s.dcs.MessageReactionAdd(chID, msgID, reaction)
}

// CommandAdd adds a new slash-command (see [discordgo.ApplicationCommand]) from
// a [Command].
func (s *Session) CommandAdd(cmd Command) error {
	if _, ok := s.Commands[cmd.Definition.Name]; ok {
		return fmt.Errorf("command with name `%s` already exists", cmd.Definition.Name)
	}

	if _, err := s.dcs.ApplicationCommandCreate(s.AppID, s.ServerID, cmd.Definition); err != nil {
		return fmt.Errorf("command creation `%s` failed: %v", cmd.Definition.Name, err)
	}

	log.Info("Command registered", "name", cmd.Definition.Name)
	s.Commands[cmd.Definition.Name] = cmd.Handler
	return nil
}

// HandlerAdd adds an event handler and associates it with the given name. Names
// must be unique to allow deleting them at a later point in time. Errors if a
// handler for the given name already exists.
func (s *Session) HandlerAdd(name string, handler any) error {
	if _, ok := s.Handlers[name]; ok {
		return fmt.Errorf("handler for name `%s` already exists", name)
	}

	rv := reflect.ValueOf(handler)
	rt := rv.Type()

	// Wrap handler to allow generic logging for all handlers.
	fn := reflect.MakeFunc(rt, func(in []reflect.Value) []reflect.Value {
		log.Info("Executing handler", "name", name)
		rv.Call(in)
		return nil
	}).Interface()

	log.Info("Handler registered", "name", name)
	s.Handlers[name] = s.dcs.AddHandler(fn)
	return nil
}

// HandlerRemove removes the event handler for the given name. Results in a noop
// if no handler exists for the name.
func (s *Session) HandlerRemove(name string) {
	if h, ok := s.Handlers[name]; ok {
		log.Debug("Handler removed", "name", name)
		h()
	}
}
