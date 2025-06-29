package main

import (
	"database/sql"
	"strings"

	"github.com/charmbracelet/log"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func NewDB() *DB {
	db, err := sql.Open("sqlite3", "stats.sqlite")
	if err != nil {
		log.Fatal("Failed to open database", "err", err)
	}

	return &DB{db}
}

func (db *DB) Setup() error {
	log.Info("Configuring database")

	// Bootstrap table for daily stats
	stmt := `
	create table
		if not exists
		today (
			user_id       string not null primary key,
			classic       int,
			quote         int,
			ability       int,
			ability_check bool,
			emoji         int,
			splash        int,
			splash_check  bool,
			elo_change    float64
		);
	`
	if _, err := db.conn.Exec(stmt); err != nil {
		log.Error("Failed to execute statement", "stmt", strings.ReplaceAll(stmt, "\t", "  "), "err", err)
		return err
	}

	return nil
}

// Gets the user's daily stats.
func (db *DB) GetStatsToday(uID string) (*StatsToday, error) {
	s := &StatsToday{&CmpStats{}, "", 0}

	row := db.conn.QueryRow("select * from today where user_id = ?", uID)
	if err := row.Scan(
		&s.UserID,
		&s.Classic,
		&s.Quote,
		&s.Ability,
		&s.AbilityCheck,
		&s.Emoji,
		&s.Splash,
		&s.SplashCheck,
		&s.EloChange,
	); err != nil {
		return nil, err
	}

	return s, nil
}

// Update user's daily stats. Primary key conflicts indicate that the user's
// stats have already been recorded.
func (db *DB) SetStatsToday(stats *StatsToday) error {
	stmt := `
	insert into
		today (
			user_id,
			classic,
			quote,
			ability,
			ability_check,
			emoji,
			splash,
			splash_check,
			elo_change
		)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	if _, err := db.conn.Exec(
		stmt,
		stats.UserID,
		stats.Classic,
		stats.Quote,
		stats.Ability,
		stats.AbilityCheck,
		stats.Emoji,
		stats.Splash,
		stats.SplashCheck,
		stats.EloChange,
	); err != nil {
		log.Error("Failed to execute statement", "stmt", strings.ReplaceAll(stmt, "\t", "  "), "stats", stats, "err", err)
		return err
	}

	return nil
}
