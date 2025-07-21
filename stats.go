package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
	"unsafe"
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

// TotalStats contains a summary of a user's cumulative LoLdle stats over all
// played games (see [LoldleStats]). This additionally includes the user's ID as
// well as their current Elo rating.
type TotalStats struct {
	UserID string `db:"id"`

	Classic      int `db:"classic"`
	Quote        int `db:"quote"`
	Ability      int `db:"ability"`
	AbilityCheck int `db:"ability_check"`
	Emoji        int `db:"emoji"`
	Splash       int `db:"splash"`
	SplashCheck  int `db:"splash_check"`

	DaysPlayed int `db:"days_played"`
	Elo        int `db:"elo"`
}

// NewTotalStats creates [TotalStats] for the given user.
func NewTotalStats(uID string) *TotalStats {
	return &TotalStats{UserID: uID, Elo: 1000}
}

func (s *TotalStats) String() string {
	name, err := session.GetUserName(s.UserID)
	if err != nil {
		return "Something went wrong :\\"
	}

	days := s.DaysPlayed
	if days == 0 {
		days = 1
	}

	return fmt.Sprintf(
		`%s
%s
Classic    %.1f
Quote      %.1f
Ability    %.1f (%.2f)
Emoji      %.1f
Splash     %.1f (%.2f)
DaysPlayed %d
Elo        %d
`,
		fmt.Sprintf("\x1b\n%s", name),
		strings.Repeat("─", utf8.RuneCountInString(name)),
		float32(s.Classic/days),
		float32(s.Quote/days),
		float32(s.Ability/days),
		float32(s.AbilityCheck/days),
		float32(s.Emoji/days),
		float32(s.Splash/days),
		float32(s.SplashCheck/days),
		s.DaysPlayed,
		s.Elo,
	)
}

// Update modifies the contained stats with the results from a game (an instance
// of [DailyStats]). This also modifies the recorded number of days played as
// well as the stored Elo rating.
func (s *TotalStats) Update(d *DailyStats) {
	s.DaysPlayed += 1
	s.Elo += d.EloChange
	if s.Elo < 0 {
		s.Elo = 0
	}

	s.Classic += d.Classic
	s.Quote += d.Quote
	s.Ability += d.Ability
	s.Emoji += d.Emoji
	s.Splash += d.Splash

	// Fast inline bool->int casting (i.e. 0 / 1). Conversion to untyped pointer
	// allows cast to *byte, which in turn allows cast to int.
	s.AbilityCheck += int(*(*byte)(unsafe.Pointer(&d.AbilityCheck)))
	s.SplashCheck += int(*(*byte)(unsafe.Pointer(&d.SplashCheck)))
}
