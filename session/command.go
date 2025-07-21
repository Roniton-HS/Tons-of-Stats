package session

import "github.com/bwmarrin/discordgo"

// Handler represents a handler for a [*discordgo.ApplicationCommand].
//
// Handlers are called with the user interaction itself (i.e.
// [*discordgo.Interaction]), not the usual [*discordgo.InteractionCreate].
type Handler func(*discordgo.Session, *discordgo.Interaction) *discordgo.InteractionResponse

// Command wraps a [*discordgo.ApplicationCommand], containing both the command
// definition itself, as well as the corresponding event handler in form of a
// [Handler] (see also [discordgo.EventHandler]).
type Command struct {
	Definition *discordgo.ApplicationCommand
	Handler    Handler
}
