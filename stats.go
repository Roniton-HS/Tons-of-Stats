package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type StatsToday struct {
	UserID string

	Classic      int
	Quote        int
	Ability      int
	AbilityCheck bool
	Emoji        int
	Splash       int
	SplashCheck  bool

	EloChange float64
}

func (s *StatsToday) String() string {
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

	return fmt.Sprintf(
		`%s
%s
Classic %d
Quote   %d
Ability %d %s
Emoji   %d
Splash  %d %s
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
	)
}

type StatsTotal struct {
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
