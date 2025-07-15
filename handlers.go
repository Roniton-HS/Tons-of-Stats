package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

// Handle LoLdle result messages and update user stats accordingly.
func recordStats(_ *discordgo.Session, msg *discordgo.MessageCreate) {
	if ch, err := session.GetChannelID(env.ResultsCh); err != nil || msg.ChannelID != ch {
		return
	} else if !CanParse(msg.Content) {
		log.Debug("Ignoring message", "user", msg.Author.ID, "msg", msg.Content)
		return
	}

	lStats, err := ParseStats(msg.Content)
	if err != nil {
		log.Error("Message parsing failed", "err", err)
		session.MsgReact(msg.ChannelID, msg.ID, "❓")
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
