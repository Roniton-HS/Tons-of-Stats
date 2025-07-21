package models

import (
	"fmt"
)

// DailyStats contains a summary of a single game of LoLdle (see [LoldleStats]).
// This additionally includes the ID of the user who posted the stats as well as
// the change in Elo resulting from the stats.
type DailyStats struct {
	UserID string `db:"id"`

	Classic      int  `db:"classic"`
	Quote        int  `db:"quote"`
	Ability      int  `db:"ability"`
	AbilityCheck bool `db:"ability_check"`
	Emoji        int  `db:"emoji"`
	Splash       int  `db:"splash"`
	SplashCheck  bool `db:"splash_check"`

	EloChange int `db:"elo_change"`
}

// NewDailyStats creates [DailyStats] for the given user with the given stats.
func NewDailyStats(uID string, l *LoldleStats) *DailyStats {
	return &DailyStats{
		UserID: uID,

		Classic:      l.Classic,
		Quote:        l.Quote,
		Ability:      l.Ability,
		AbilityCheck: l.AbilityCheck,
		Emoji:        l.Emoji,
		Splash:       l.Splash,
		SplashCheck:  l.SplashCheck,

		EloChange: l.CalculateElo(),
	}
}

func (s *DailyStats) String() string {
	aChk, sChk := "", ""
	if s.AbilityCheck {
		aChk = "✔"
	}
	if s.SplashCheck {
		sChk = "✔"
	}

	elo := "+"
	if s.EloChange < 0 {
		elo = "-"
	}

	return fmt.Sprintf(
		`
Classic    %d
Quote      %d
Ability    %d %s
Emoji      %d
Splash     %d %s
EloChange  %s%d
`,
		s.Classic,
		s.Quote,
		s.Ability,
		aChk,
		s.Emoji,
		s.Splash,
		sChk,
		elo,
		s.EloChange,
	)
}
