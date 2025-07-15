package main

import (
	"database/sql"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

// Handle LoLdle result messages and update user stats accordingly.
func RecordStats(_ *discordgo.Session, msg *discordgo.MessageCreate) {
	if ch, err := session.GetChannelID(env.ResultsCh); err != nil || msg.ChannelID != ch {
		log.Debug("Ignoring message", "uID", msg.Author.ID, "msg", msg.Content, "err", err)
		return
	} else if !CanParse(msg.Content) {
		log.Debug("Ignoring message", "uID", msg.Author.ID, "msg", msg.Content, "reason", "not parsable")
		return
	}

	parsed, err := ParseStats(msg.Content)
	if err != nil {
		log.Error("Message parsing failed", "err", err)
		session.MsgReact(msg.ChannelID, msg.ID, "❓")
		return
	}

	stats := NewDailyStats(msg.Author.ID, parsed)
	if err := updateStats(stats); err != nil {
		session.MsgReact(msg.ChannelID, msg.ID, "❌")
	} else {
		session.MsgReact(msg.ChannelID, msg.ID, "✅")
	}
}

// Updates the user's daily and total stats.
//
// WARN: Current implementation is not transactional and may leave database in a
// broken state!
func updateStats(daily *DailyStats) error {
	log.Info("Updating daily stats", "uID", daily.UserID, "stats", daily)
	if err := db.Today.Update(daily.UserID, daily); err != nil {
		log.Warn("Update failed", "uID", daily.UserID, "err", err)
		return err
	}

	total, err := db.Total.Get(daily.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			total = NewTotalStats(daily.UserID)
		} else {
			log.Error("Failed to fetch", "table", "total", "err", err)
			return err
		}
	}

	// Total stats can safely be updated here, since any violations (e.g. from
	// multiple submissions) are caught during the first update.
	total.Update(daily)

	log.Info("Updating total stats", "uID", daily.UserID, "stats", total)
	if err := db.Total.Update(daily.UserID, total); err != nil {
		log.Warn("Update failed", "uID", total.UserID, "err", err)
		return err
	}

	return nil
}
