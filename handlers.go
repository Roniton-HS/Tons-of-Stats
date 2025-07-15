package main

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

// Handle LoLdle result messages and update user stats accordingly.
func recordStats(_ *discordgo.Session, msg *discordgo.MessageCreate) {
	if ch, err := session.GetChannelID("result-spam"); err != nil || msg.ChannelID != ch {
		return
	} else if !strings.HasPrefix(msg.Content, LoldleHeader) {
		// TODO: error message
		return
	}

	lStats, err := ParseStats(msg.Content)
	if err != nil {
		log.Error("Message parsing failed", "err", err)
		// TODO: error message
		return
	}

	// TODO:
	// + elo calculation
	// + update total stats
	sToday := &StatsToday{
		msg.Author.ID,
		lStats.Classic,
		lStats.Quote,
		lStats.Ability,
		lStats.AbilityCheck,
		lStats.Emoji,
		lStats.Splash,
		lStats.SplashCheck,
		0,
	}

	if err := db.Today.Update(msg.Author.ID, sToday); err != nil {
		log.Warn("Failed to record daily stats", "user", msg.Author.ID, "msg", msg.Content, "err", err)
		session.MsgReact(msg.ChannelID, msg.ID, "❌")
	} else {
		log.Info("Daily stats recorded", "user", msg.Author.ID)
		session.MsgReact(msg.ChannelID, msg.ID, "✅")
	}
}

// Handle requests for stat-display.
//
// TODO: refactor to proper command
func displayStats(_ *discordgo.Session, msg *discordgo.MessageCreate) {
	if strings.TrimSpace(msg.Content) != "stats" {
		return
	}

	stats, err := db.Today.Get(msg.Author.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			session.MsgSend(msg.ChannelID, "No stats recorded today.")
			return
		}

		log.Warn("Stat retrieval failed", "chID", msg.ChannelID, "uID", msg.Author.ID, "err", err)
		session.MsgSend(msg.ChannelID, "Failed to retrieve your stats.")
		return
	}

	session.MsgSend(msg.ChannelID, stats.String())
}
