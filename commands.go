package main

import (
	"database/sql"
	"errors"
	"fmt"
	sess "tons-of-stats/session"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

// List of all application commands to register at startup.
var cmds = []sess.Command{
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
					msg = "❌  **No stats recorded.**"
				} else {
					log.Warn("Stat retrieval failed", "chID", i.ChannelID, "uID", i.Member.User.ID, "err", err)
					msg = fmt.Sprintf(
						"❌  **%s**\n-# %s",
						"Could not retrieve your stats. Please try again.",
						"If this error persists, please contact the moderation team.",
					)
				}
			} else {
				msg = fmt.Sprintf("## %s\n```ansi\n%s\n```", "Daily stats:", stats.String())
			}

			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: sess.IS_COMPONENTS_V2 ^ discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.Container{
							AccentColor: &ACCENT,
							Components: []discordgo.MessageComponent{
								discordgo.TextDisplay{
									Content: msg,
								},
							},
						},
					},
				},
			}
		},
	},
}
