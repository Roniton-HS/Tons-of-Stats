package main

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/charmbracelet/log"
)

type Category struct {
	Key     []byte
	Value   []byte
	Checked bool
}

type CmpStats struct {
	Classic      int
	Quote        int
	Ability      int
	AbilityCheck bool
	Emoji        int
	Splash       int
	SplashCheck  bool
}

func (c CmpStats) String() string {
	return fmt.Sprintf(`Classics: %d
Quote: %d
Ability: %d
Emoji: %d
Splash: %d`,
		c.Classic,
		c.Quote,
		c.Ability,
		c.Emoji,
		c.Splash,
	)
}

func ParseStats(msg string) (*CmpStats, error) {
	mErr, cErr := errors.New("malformed message"), errors.New("internal conversion error")
	lines := bytes.Split([]byte(msg), []byte("\n"))

	if !bytes.Equal(lines[0], []byte("I've completed all the modes of #LoLdle today:")) {
		log.Warn("Invalid message start sequence", "msg", msg, "seq", lines[0])
		return nil, mErr
	}

	// Parse message into category-slice before conversion
	categories := make([]Category, 0, 5)

	for i, ln := range lines[1:] {
		log.Debug(fmt.Sprintf("Parsing line %d", i), "ln", string(ln))

		f := bytes.Fields(ln)
		if len(f) < 3 {
			log.Warn("Invalid message segment", "ln", string(ln))
			return nil, mErr
		}

		key, value := f[1], f[2]
		checked := len(f) > 3 && bytes.Equal(f[3], []byte("✓"))

		categories = append(categories, Category{key[:len(key)-1], value, checked})
	}

	// Validate and convert parsed slice to proper struct
	rv := reflect.ValueOf(&CmpStats{})
	for i, c := range categories {
		log.Debug(fmt.Sprintf("Parsing category %d", i), "category", c)

		f := reflect.Indirect(rv).FieldByName(string(c.Key))
		if !f.IsValid() {
			log.Warn("Message contains invalid category", "category", string(c.Key))
			return nil, mErr
		}

		// Validate and set field's value
		if bytes.ContainsFunc(c.Value, func(r rune) bool {
			return r < 48 || r > 57 // rune is outside of valid ASCII range
		}) {
			log.Warn("Illegal value", "category", string(c.Key), "value", string(c.Value))
			return nil, mErr
		}

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

		// Set "<Field>Check" if necessary
		if c.Checked {
			cf := reflect.Indirect(rv).FieldByName(fmt.Sprintf("%sCheck", string(c.Key)))
			cf.Set(reflect.ValueOf(c.Checked)) // always `true` at this point
		}
	}

	stats, ok := rv.Elem().Interface().(CmpStats)
	if !ok {
		log.Error("Conversion failed for `reflect.Value`")
		return nil, cErr
	}

	log.Debug("Message parsed", "stats", stats)
	return &stats, nil
}
