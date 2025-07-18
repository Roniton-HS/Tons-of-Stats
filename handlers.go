package main

import (
	"database/sql"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

// RecordStats acts as an event handler (see [discordgo.EventHandler]) to record
// information about newly played LoLdle games. Irrelevant messages (i.e.
// messages not containing game stats) are ignored.
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

	if !CanParse(msg.Content) {
		log.Debug("Ignoring message", "uID", msg.Author.ID, "msg", msg.Content, "reason", "not parsable")
		return
	}

	// Parse message as [DailyStats] for the message's author.
	parsed, err := ParseStats(msg.Content)
	if err != nil {
		log.Error("Message parsing failed", "err", err)
		session.MsgReact(msg.ChannelID, msg.ID, "❓")
		return
	}

	// Update daily and total stats for the message's author.
	stats := NewDailyStats(msg.Author.ID, parsed)
	if err := updateStats(stats); err != nil {
		session.MsgReact(msg.ChannelID, msg.ID, "❌")
	} else {
		session.MsgReact(msg.ChannelID, msg.ID, "✅")
	}
}

// updateStats modifies the user's daily and total stats with the given stats.
func updateStats(daily *DailyStats) error {
	log.Info("Updating daily stats", "uID", daily.UserID, "stats", daily)

	err := dal.DB.Transaction(func(tx Tx) error {
		// Update daily stats if possible. Primary key conflicts indicate duplicate
		// submissions within the same day.
		txToday := dal.Today.WithTx(tx)
		if err := txToday.Create(daily.UserID, daily); err != nil {
			return err
		}

		log.Info("Fetching total stats", "uID", daily.UserID)

		// Get user's total stats or create new [TotalStats] if it's their first
		// time playing.
		txTotal := dal.Total.WithTx(tx)
		total, err := txTotal.Get(daily.UserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				total = NewTotalStats(daily.UserID)
			} else {
				return err
			}
		}

		log.Info("Updating total stats", "uID", daily.UserID, "stats", total)

		// Total stats can safely be updated here, since any violations (e.g. from
		// multiple submissions) are caught during the first update.
		total.Update(daily)
		if err := txTotal.Update(daily.UserID, total); err != nil {
			return err
		}

		return nil
	})

	return err
}
