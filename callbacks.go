package main

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"tons-of-stats/db"
	"tons-of-stats/models"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

// updateStats modifies the user's daily and total stats with the given stats.
func updateStats(daily *models.DailyStats) error {
	log.Info("Updating daily stats", "uID", daily.UserID, "stats", daily)

	err := dal.DB.Transaction(func(tx db.Tx) error {
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
				log.Info("No stats found - creating total stats", "uID", daily.UserID)
				total = models.NewTotalStats(daily.UserID)

				if err := txTotal.Create(total.UserID, total); err != nil {
					return err
				}
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

// updateLeaderboard updates the global leaderboard information.
func updateLeaderboard() error {
	log.Info("Updating leaderboard")
	chID, err := session.GetChannelID(env.StatsCh)
	if err != nil {
		return err
	}

	stats, err := dal.Total.GetAll()
	if err != nil {
		return err
	}

	// PERF: DB ordering
	slices.SortFunc(stats, func(a *models.TotalStats, b *models.TotalStats) int {
		if a.Elo < b.Elo {
			return -1
		} else if a.Elo > b.Elo {
			return 1
		}

		return 0
	})

	var sb strings.Builder
	sb.WriteString("## Rank Ladder\n\n")

	for i, s := range stats {
		sb.WriteString(printRank(i, s))
	}

	cmp := []discordgo.MessageComponent{
		discordgo.Container{
			AccentColor: &ACCENT,
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{
					Content: sb.String(),
				},
			},
		},
	}
	return session.MsgSendComplex(chID, cmp)
}

// printRank formats a users rank and stats for display in the leaderboard.
func printRank(i int, s *models.TotalStats) string {
	name, err := session.GetUserName(s.UserID)
	if err != nil {
		log.Warn("Failed to resolve name", "uID", s.UserID, "err", err)
		name = "!?unknown"
	}

	var prefix = "\x1b[30m-"
	var change = 0

	// PERF: pre-fetch all / DB correlation
	if daily, err := dal.Today.Get(s.UserID); err == nil {
		if daily.EloChange > 0 {
			prefix = "\x1b[32m+"
		} else if daily.EloChange < 0 {
			prefix = "\x1b[31m-"
		}
		change = daily.EloChange
	}

	return fmt.Sprintf("```ansi\n#%d %s \x1b[30m|\x1b[0m \x1b[1m%d\x1b[0m Elo [%s%d\x1b[0m]\n```", i+1, name, s.Elo, prefix, change)
}
