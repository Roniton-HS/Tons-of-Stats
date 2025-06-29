package main

import (
	"database/sql"
	"strings"

	"github.com/charmbracelet/log"
	_ "github.com/mattn/go-sqlite3"
)

// Represents a connection to a table in the database for which a connection is
// held. Tables are abstracted behind the interface to provide uniform access to
// different tables with potentially different types for query parameters or
// results.
type Table[T any] interface {
	Get(id string) (T, error)
	GetAll() ([]T, error)
	Update(id string, t T) error
}

// Groups and exposes multiple connections to the same underlying database.
type StatsDB struct {
	db *sql.DB // Main database handle - used by contained connections

	Today Table[*StatsToday]
}

func NewStatsDB(db *sql.DB) *StatsDB {
	return &StatsDB{db, &TblToday{db}}
}

func (s *StatsDB) Close() {
	s.db.Close()
}

func (s *StatsDB) Setup() error {
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
	if _, err := s.db.Exec(stmt); err != nil {
		log.Error("Failed to execute statement", "stmt", strings.ReplaceAll(stmt, "\t", "  "), "err", err)
		return err
	}

	return nil
}

// Represents a connection to the `today` table.
type TblToday struct {
	db *sql.DB
}

func (tbl *TblToday) Get(id string) (*StatsToday, error) {
	st := &StatsToday{&CmpStats{}, "", 0}

	row := tbl.db.QueryRow("select * from today where user_id = ?", id)
	if err := row.Scan(
		&st.UserID,
		&st.Classic,
		&st.Quote,
		&st.Ability,
		&st.AbilityCheck,
		&st.Emoji,
		&st.Splash,
		&st.SplashCheck,
		&st.EloChange,
	); err != nil {
		return nil, err
	}

	return st, nil
}

func (tbl *TblToday) GetAll() ([]*StatsToday, error) {
	rows, err := tbl.db.Query("select * from today")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*StatsToday
	for rows.Next() {
		s := &StatsToday{&CmpStats{}, "", 0}

		if err := rows.Scan(
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
	}

	return stats, nil
}

// Update user's daily stats. Primary key conflicts indicate that the user's
// stats have already been recorded.
func (tbl *TblToday) Update(id string, t *StatsToday) error {
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

	if _, err := tbl.db.Exec(
		stmt,
		t.UserID,
		t.Classic,
		t.Quote,
		t.Ability,
		t.AbilityCheck,
		t.Emoji,
		t.Splash,
		t.SplashCheck,
		t.EloChange,
	); err != nil {
		log.Error("Failed to execute statement", "stmt", strings.ReplaceAll(stmt, "\t", "  "), "entity", t, "err", err)
		return err
	}

	return nil
}
