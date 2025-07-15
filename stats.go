package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Calculates the change in user Elo resulting from the given stats.
// Elo gains are based on the distribution table below, where columns correspond
// to the number of guesses for each category.
//
//	|         |  1 |  2 |  3 |  4 |  5 |
//	| ------- | -: | -: | -: | -: | -: |
//	| Classic | +4 | +4 | +2 | -2 | -4 |
//	| Quote   | +4 | +2 | -2 | -4 |    |
//	| Ability | +4 | -2 | -4 |    |    |
//	| Emoji   | +4 | +2 | -2 | -4 |    |
//	| Splash  | +4 | +2 | -2 | -4 |    |
//
// Categories with more guesses than listed net -4 Elo each.
// Guessing the ability or splash art correctly nets +2 Elo each.
func CalculateElo(l *LoldleStats) int {
	var elo int

	// Classic
	if l.Classic <= 5 {
		elo += 4
		if l.Classic > 2 { // "grace"-guess requires guard
			elo -= 2 * (l.Classic - 1)
		}
	} else {
		elo -= 4
	}

	// Quote
	if l.Quote <= 4 {
		elo += 4
		elo -= 2 * (l.Quote - 1)
	} else {
		elo -= 4
	}

	// Ability
	switch l.Ability {
	case 1:
		elo += 4
	case 2:
		elo -= 2
	default:
		elo -= 4
	}

	// Emoji
	if l.Emoji <= 4 {
		elo += 4
		elo -= 2 * (l.Emoji - 1)
	} else {
		elo -= 4
	}

	// Splash
	if l.Splash <= 4 {
		elo += 4
		elo -= 2 * (l.Splash - 1)
	} else {
		elo -= 4
	}

	// Checkmarks
	if l.AbilityCheck {
		elo += 2
	}
	if l.SplashCheck {
		elo += 2
	}

	return elo
}

type DailyStats struct {
	UserID string

	Classic      int
	Quote        int
	Ability      int
	AbilityCheck bool
	Emoji        int
	Splash       int
	SplashCheck  bool

	EloChange int
}

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

		EloChange: CalculateElo(l),
	}
}

func (s *DailyStats) String() string {
	name, err := session.GetUserName(s.UserID)
	if err != nil {
		return "Something went wrong :\\"
	}

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
		`%s
%s
Classic    %d
Quote      %d
Ability    %d %s
Emoji      %d
Splash     %d %s
EloChange  %s%d
`,
		fmt.Sprintf("\x1b\n%s", name),
		strings.Repeat("─", utf8.RuneCountInString(name)),
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

type TotalStats struct {
	UserID string

	Classic      int
	Quote        int
	Ability      int
	AbilityCheck int
	Emoji        int
	Splash       int
	SplashCheck  int

	DaysPlayed int
	Elo        float64
}
