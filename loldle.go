package main

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/charmbracelet/log"
)

const LoldleHeader = "I've completed all the modes of #LoLdle today:"

// Category represents the score for a single LoLdle category.
type Category struct {
	Key     []byte // Category name
	Value   []byte // Category score
	Checked bool   // Whether the category has an additional success-check
}

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

// CanParse reports whether the given message may be parsed into [LoldleStats].
//
// This function only performs baseline validation, such that parsing for a
// given message may fail, even if CanParse returns true. The opposite is not
// true; i.e. if CanParse returns false, the message can definitely not be
// parsed as [LoldleStats].
func CanParse(msg string) bool {
	lines := bytes.Split([]byte(msg), []byte("\n"))

	// msg must contain all five LoLdle categories as well as an additional header
	// (see [LoldleHeader]).
	if len(lines) < 6 {
		log.Warn("Message too short", "msg", msg, "want", 6, "got", len(lines))
		return false
	}

	if !bytes.Equal(lines[0], []byte(LoldleHeader)) {
		log.Warn("Invalid message start sequence", "msg", msg, "seq", string(lines[0]))
		return false
	}

	return true
}

// ParseStats tries to create a new [LoldleStats] from msg.
func ParseStats(msg string) (*LoldleStats, error) {
	mErr, cErr := errors.New("malformed message"), errors.New("internal conversion error")
	if !CanParse(msg) {
		return nil, mErr
	}

	lines := bytes.Split([]byte(msg), []byte("\n"))

	// Parse message into categories before conversion
	categories := make([]Category, 0, 5)

	// First line must be [LoldleHeader]. We assume no empty lines, such that the
	// next five lines must be the category results.
	for i, ln := range lines[1:6] {
		log.Debug(fmt.Sprintf("Parsing line %d", i), "ln", string(ln))

		f := bytes.Fields(ln)
		// Each line must consist of an emoji, the category's name and value, with
		// an optional checkmark.
		if len(f) < 3 {
			log.Warn("Invalid message segment", "ln", string(ln))
			return nil, mErr
		}

		key, value := f[1], f[2]
		checked := len(f) > 3 && bytes.Equal(f[3], []byte("✓"))

		categories = append(categories, Category{
			key[:len(key)-1], // Remove trailing `:`
			value,
			checked,
		})
	}

	if len(categories) != 5 {
		log.Warn("Invalid category length", "want", 5, "got", len(categories))
		return nil, mErr
	}

	// Validate parsed categories and parse to [LoldleStats]. The validation uses
	// the struct fields defined on [LoldleStats].
	rv := reflect.ValueOf(&LoldleStats{})
	for i, c := range categories {
		log.Debug(fmt.Sprintf("Parsing category %d", i), "category", c)

		// Category name validation.
		f := reflect.Indirect(rv).FieldByName(string(c.Key))
		if !f.IsValid() {
			log.Warn("Message contains invalid category", "category", string(c.Key))
			return nil, mErr
		}

		// Category value validation.
		if bytes.ContainsFunc(c.Value, func(r rune) bool {
			return r < 48 || r > 57 // r is outside of the valid ASCII range
		}) {
			log.Warn("Illegal value", "category", string(c.Key), "value", string(c.Value))
			return nil, mErr
		}

		// Convert and set value on output struct.
		iv, err := strconv.Atoi(string(c.Value))
		if err != nil {
			log.Warn("Conversion failed", "category", string(c.Key), "value", string(c.Value), "err", err)
			return nil, cErr
		}
		if iv == 0 { // negative values are filtered by checking ASCII-range (see above)
			log.Warn("Illegal value", "category", string(c.Key), "value", iv)
			return nil, mErr
		}
		f.Set(reflect.ValueOf(iv))

		// Set "<Field>Check" if necessary.
		if c.Checked {
			cf := reflect.Indirect(rv).FieldByName(fmt.Sprintf("%sCheck", string(c.Key)))
			cf.Set(reflect.ValueOf(c.Checked)) // c.Checked = true
		}
	}

	// Retrieve concrete value from reflected struct.
	stats, ok := rv.Elem().Interface().(LoldleStats)
	if !ok {
		log.Error("Conversion failed for `reflect.Value`")
		return nil, cErr
	}

	log.Debug("Message parsed", "stats", stats)
	return &stats, nil
}
