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
	crs := "\x1b[1;31m✗\x1b[0m"
	chk := "\x1b[1;32m✓\x1b[0m"

	aChk, sChk := crs, crs
	if s.AbilityCheck {
		aChk = chk
	}
	if s.SplashCheck {
		sChk = chk
	}

	elo := "\x1b[1;32m+"
	if s.EloChange < 0 {
		elo = "\x1b[1;31m-"
	}

	return fmt.Sprintf(
		`
Classic  %2d
Quote    %2d
Ability  %2d %s
Emoji    %2d
Splash   %2d %s
Elo     %s%2d%s
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
		"\x1b[0m",
	)
}
