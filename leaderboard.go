package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"tons-of-stats/models"
	sess "tons-of-stats/session"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

var ErrNoMsg = errors.New("leaderboard: no suitable message found")
var lbHeader = "## Leaderboard"
var favicon = "https://loldle.net/favicon.ico"

type Leaderboard struct {
	dal     *DAL
	env     *Env
	session *sess.Session

	// Channel ID to use for retrieving and posting messages.
	chID string
	// Message ID of the message displaying the leaderboard.
	msgID string
}

// NewLeaderboard creates a new Leaderboard.
func NewLeaderboard(dal *DAL, env *Env, session *sess.Session) (*Leaderboard, error) {
	chID, err := session.GetChannelID(env.StatsCh)
	if err != nil {
		return nil, err
	}

	msgID, err := findMsg(session, chID)
	if err != nil {
		if errors.Is(err, ErrNoMsg) {
			msgID, err = createMsg(session, chID)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		log.Info("Reusing existing leaderboard", "msgID", msgID)
	}

	return &Leaderboard{dal, env, session, chID, msgID}, nil
}

// Update updates the leaderboard with the currently available user stats to
// reflect any potential changes.
func (l *Leaderboard) Update() error {
	log.Info("Updating leaderboard", "chID", l.chID, "msgID", l.msgID)
	if err := l.invalidateMsg(); err != nil {
		log.Warn("Update failed", "chID", l.chID, "msgID", l.msgID, "err", err)
		return err
	}

	// PERF: prefetch + cache
	stats, err := l.dal.Total.GetAll()
	if err != nil {
		return err
	}

	var pRank, pName, pElo []string
	var rank, name, elo []string
	if len(stats) > 3 {
		pRank, pName, pElo = fmtStats(stats[:3])
		rank, name, elo = fmtStats(stats[3:])
	} else {
		pRank, pName, pElo = fmtStats(stats)
	}

	embeds := []*discordgo.MessageEmbed{
		{
			Title:       "Podium",
			Description: fmt.Sprintf("-# Last Update: %s", time.Now().Format(time.DateOnly+" at "+time.Kitchen)),
			Color:       ACCENT,
			// FIX: image shows up for one frame, then disappears. Potentially
			// relevant: discord/discord-api-docs/issues/6171.
			Thumbnail: &discordgo.MessageEmbedThumbnail{URL: favicon},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Rank",
					Value:  strings.Join(pRank, "\n"),
					Inline: true,
				},
				{
					Name:   "Name",
					Value:  strings.Join(pName, "\n"),
					Inline: true,
				},
				{
					Name:   "Elo",
					Value:  strings.Join(pElo, "\n"),
					Inline: true,
				},
			},
		},
	}

	// TODO: pagination
	if len(stats) > 3 {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title: "Ranked Ladder",
			Color: ACCENT,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Rank",
					Value:  strings.Join(rank, "\n"),
					Inline: true,
				},
				{
					Name:   "Name",
					Value:  strings.Join(name, "\n"),
					Inline: true,
				},
				{
					Name:   "Elo",
					Value:  strings.Join(elo, "\n"),
					Inline: true,
				},
			},
		},
		)
	}

	edit := &discordgo.MessageEdit{
		Channel: l.chID,
		ID:      l.msgID,
		Content: &lbHeader,
		Embeds:  &embeds,
	}
	if _, err := l.session.MsgEditComplex(edit); err != nil {
		log.Warn("Update failed", "chID", l.chID, "msgID", l.msgID, "err", err)
		return err
	}

	log.Debug("Update complete", "chID", l.chID, "msgID", l.msgID)
	return nil
}

// invalidateMsg ensures the message for the stored msgID still points to a
// valid message and performs corrective measures in case it doesn't. This is
// required in cases where the original leaderboard message is deleted while the
// application is running.
func (l *Leaderboard) invalidateMsg() error {
	if _, err := session.MsgGet(l.chID, l.msgID); err == nil {
		return nil
	} else {
		log.Warn("Invalid leaderboard message", "chID", l.chID, "msgID", l.msgID, "err", err)
	}

	msgID, err := createMsg(l.session, l.chID)
	if err != nil {
		return err
	}

	l.msgID = msgID
	return nil
}

// findMsg tries to find a pre-existing leaderboard message that can be reused.
func findMsg(session *sess.Session, chID string) (msgID string, err error) {
	msgs, err := session.MsgList(chID)
	if err != nil {
		log.Warn("Failed to retrieve messages", "chID", chID, "err", err)
		return "", err
	}

	for _, m := range msgs {
		if m.Author.ID != session.AppID || m.Content != lbHeader {
			continue
		}

		return m.ID, nil
	}

	return "", ErrNoMsg
}

// createMsg creates a new message to use as a leaderboard.
func createMsg(session *sess.Session, chID string) (msgID string, err error) {
	log.Info("Creating new leaderboard")

	m, err := session.MsgSendComplex(chID, &discordgo.MessageSend{Content: lbHeader})
	if err != nil {
		log.Error("Creation failed", "err", err)
		return "", err
	}

	log.Debug("Creation complete", "msgID", m.ID)
	return m.ID, nil
}

// fmtStats orders and formats all user stats for display in the leaderboard.
func fmtStats(stats []*models.TotalStats) (rank []string, name []string, elo []string) {
	// DB ordering
	slices.SortFunc(stats, func(a *models.TotalStats, b *models.TotalStats) int {
		if a.Elo < b.Elo {
			return 1
		} else if a.Elo > b.Elo {
			return -1
		}

		return 0
	})

	rank = make([]string, 0, len(stats))
	name = make([]string, 0, len(stats))
	elo = make([]string, 0, len(stats))

	for i, s := range stats {
		user, err := session.GetUserName(s.UserID)
		if err != nil {
			log.Warn("Failed to resolve name", "uID", s.UserID, "err", err)
			user = "!?unknown"
		}

		var prefix = "\x1b[30m+"
		var change = 0

		// PERF: prefetch / DB correlation
		if daily, err := dal.Today.Get(s.UserID); err == nil {
			if daily.EloChange > 0 {
				prefix = "\x1b[32m+"
			} else if daily.EloChange < 0 {
				prefix = "\x1b[31m-"
			}
			change = daily.EloChange
		}

		rank = append(rank, fmt.Sprintf("``` %d ```", i+1))
		name = append(name, fmt.Sprintf("``` %s ```", user))
		elo = append(elo, fmt.Sprintf("```ansi\n%4d [%s%d\x1b[0m]```", s.Elo, prefix, change))
	}

	return rank, name, elo
}
