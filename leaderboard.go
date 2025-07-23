package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"tons-of-stats/models"
	sess "tons-of-stats/session"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

var ErrNoMsg = errors.New("leaderboard: no suitable message found")

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
	if err := l.validateMsg(); err != nil {
		log.Warn("Update failed", "chID", l.chID, "msgID", l.msgID, "err", err)
		return err
	}

	// PERF: prefetch + cache?
	stats, err := l.dal.Total.GetAll()
	if err != nil {
		return err
	}

	cmp := []discordgo.MessageComponent{
		discordgo.Container{
			AccentColor: &ACCENT,
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{
					Content: fmt.Sprintf("## Rank Ladder\n\n%s", fmtStats(stats)),
				},
			},
		},
	}

	_, err = l.session.MsgEditComplex(l.chID, l.msgID, cmp)
	if err != nil {
		log.Warn("Update failed", "chID", l.chID, "msgID", l.msgID, "err", err)
		return err
	}

	log.Debug("Update complete", "chID", l.chID, "msgID", l.msgID)
	return nil
}

// validateMsg checks whether stored msgID still points to a valid message and
// performs corrective measures if it doesn't. This is required in cases where
// the original leaderboard message is deleted while the application is running.
func (l *Leaderboard) validateMsg() error {
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
		if m.Author.ID != session.AppID || m.Flags != sess.IS_COMPONENTS_V2 {
			continue
		}

		// TODO: check message header components?
		return m.ID, nil
	}

	return "", ErrNoMsg
}

// createMsg creates a new message to use as a leaderboard.
func createMsg(session *sess.Session, chID string) (msgID string, err error) {
	log.Info("Creating new leaderboard")

	m, err := session.MsgSendComplex(chID, []discordgo.MessageComponent{})
	if err != nil {
		log.Error("Creation failed", "err", err)
		return "", err
	}

	log.Debug("Creation complete", "msgID", m.ID)
	return m.ID, nil
}

// fmtStats orders and formats all user stats for display in the leaderboard.
func fmtStats(stats []*models.TotalStats) string {
	// DB ordering
	slices.SortFunc(stats, func(a *models.TotalStats, b *models.TotalStats) int {
		if a.Elo < b.Elo {
			return -1
		} else if a.Elo > b.Elo {
			return 1
		}

		return 0
	})

	var sb strings.Builder
	for i, s := range stats {
		sb.WriteString(fmtRank(i, s))
	}

	return sb.String()
}

// fmtRank formats a user's rank and stats for display in the leaderboard.
func fmtRank(i int, s *models.TotalStats) string {
	name, err := session.GetUserName(s.UserID)
	if err != nil {
		log.Warn("Failed to resolve name", "uID", s.UserID, "err", err)
		name = "!?unknown"
	}

	var prefix = "\x1b[30m-"
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

	return fmt.Sprintf("```ansi\n#%d %s \x1b[30m|\x1b[0m \x1b[1m%d\x1b[0m Elo [%s%d\x1b[0m]\n```", i+1, name, s.Elo, prefix, change)
}
