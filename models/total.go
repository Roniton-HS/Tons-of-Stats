package models

import (
	"fmt"
	"unsafe"
)

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
	days := s.DaysPlayed
	if days == 0 {
		days = 1
	}

	return fmt.Sprintf(
		`
Classic    %.1f
Quote      %.1f
Ability    %.1f (%.2f)
Emoji      %.1f
Splash     %.1f (%.2f)
DaysPlayed %d
Elo        %d
`,
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
