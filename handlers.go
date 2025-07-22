package main

import (
	"tons-of-stats/models"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

// RecordStats records information about newly played LoLdle games.
//
// [discordgo.EventHandler]
func RecordStats(dcs *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.ID == session.AppID {
		log.Debug("Ignoring own message", "msgID", msg.ID)
		return
	}

	if ch, err := session.GetChannelID(env.ResultsCh); err != nil {
		log.Debug("Ignoring message", "uID", msg.Author.ID, "err", err)
		return
	} else if ch != msg.ChannelID {
		log.Debug("Ignoring message", "uID", msg.Author.ID, "targetlChID", ch, "msgChID", msg.ChannelID)
		return
	}

	if !models.CanParse(msg.Content) {
		log.Debug("Ignoring message", "uID", msg.Author.ID, "msg", msg.Content, "reason", "not parsable")
		return
	}

	// Parse message as [DailyStats] for the message's author.
	parsed, err := models.ParseStats(msg.Content)
	if err != nil {
		log.Error("Message parsing failed", "err", err)
		session.MsgReact(msg.ChannelID, msg.ID, "❓")
		return
	}

	// Update daily and total stats for the message's author.
	stats := models.NewDailyStats(msg.Author.ID, parsed)
	if err := updateStats(stats); err != nil {
		session.MsgReact(msg.ChannelID, msg.ID, "❌")
	} else {
		session.MsgReact(msg.ChannelID, msg.ID, "✅")
		if err := updateLeaderboard(); err != nil {
			log.Warn("Failed to update leaderboard", "err", err)
		}
	}
}
