package main

import (
	"database/sql"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

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

// List of all application commands to register at startup.
var cmds = []Command{
	{
		Definition: &discordgo.ApplicationCommand{
			Name:        "stats",
			Description: "Returns your current daily stats, if any have been recorded.",
		},
		Handler: func(s *discordgo.Session, i *discordgo.Interaction) *discordgo.InteractionResponse {
			if i.Member == nil {
				return nil
			}

			var msg string

			// Fetch current daily stats for the member invoking the command.
			if stats, err := dal.Today.Get(i.Member.User.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					msg = "No stats recorded today."
				} else {
					log.Warn("Stat retrieval failed", "chID", i.ChannelID, "uID", i.Member.User.ID, "err", err)
					msg = "Could not retrieve your stats."
				}
			} else {
				msg = stats.String()
			}

			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: msg},
			}
		},
	},
}
