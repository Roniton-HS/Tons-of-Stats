package models

const LoldleHeader = "I've completed all the modes of #LoLdle today:"

// LoLdleStats contains a summary of a single game of LoLdle, including the
// scores for all categories and information about bonuses (i.e. checkmarks).
type LoldleStats struct {
	Classic      int
	Quote        int
	Ability      int
	AbilityCheck bool
	Emoji        int
	Splash       int
	SplashCheck  bool
}

// CalculateElo calculates the change in user Elo resulting from the given
// stats. Elo gains are based on the distribution table below, where columns
// correspond to the number of guesses for each category.
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
func (l *LoldleStats) CalculateElo() int {
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
