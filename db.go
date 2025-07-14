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
	Delete(id string) error
	DeleteAll() error
}

// Groups and exposes multiple connections to the same underlying database.
type StatsDB struct {
	db *sql.DB // Main database handle - used by contained connections

	Today Table[*StatsToday]
	Total Table[*StatsTotal]
}

func NewStatsDB(db *sql.DB) *StatsDB {
	return &StatsDB{db, &TblToday{db}, &TblTotal{db}}
}

func (s *StatsDB) Close() {
	s.db.Close()
}

func (s *StatsDB) Setup() error {
	log.Info("Configuring database")
	var stmt string

	// Bootstrap table for daily stats
	stmt = `
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

	// Bootstrap table for cumulative stats
	stmt = `
	create table
		if not exists
		total (
			user_id       string not null primary key,
			classic       int,
			quote         int,
			ability       int,
			ability_check int,
			emoji         int,
			splash        int,
			splash_check  int,
			days_played   int,
			elo           float64
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
	s := &StatsToday{}

	row := tbl.db.QueryRow("select * from today where user_id = ?", id)
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

func (tbl *TblToday) GetAll() ([]*StatsToday, error) {
	rows, err := tbl.db.Query("select * from today")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*StatsToday
	for rows.Next() {
		s := &StatsToday{}

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

		stats = append(stats, s)
	}

	return stats, nil
}

// Updates the daily stats for the user with UserID `id`.
// Primary key conflicts indicate that the user's stats have already been
// recorded.
func (tbl *TblToday) Update(id string, t *StatsToday) error {
	stmt := `
	insert into
		today
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

func (tbl *TblToday) Delete(id string) error {
	stmt := `delete from today where user_id = ?`

	if _, err := tbl.db.Exec(stmt, id); err != nil {
		log.Error("Failed to execute statement", "stmt")
		return err
	}

	return nil
}

func (tbl *TblToday) DeleteAll() error {
	stmt := `delete from today`

	if _, err := tbl.db.Exec(stmt); err != nil {
		log.Error("Failed to execute statement", "stmt")
		return err
	}

	return nil
}

// Represents a connection to the `total` table.
type TblTotal struct {
	db *sql.DB
}

func (tbl *TblTotal) Get(id string) (*StatsTotal, error) {
	s := &StatsTotal{}

	row := tbl.db.QueryRow("select * from total where user_id = ?", id)
	if err := row.Scan(
		&s.UserID,
		&s.Classic,
		&s.Quote,
		&s.Ability,
		&s.AbilityCheck,
		&s.Emoji,
		&s.Splash,
		&s.SplashCheck,
		&s.DaysPlayed,
		&s.Elo,
	); err != nil {
		return nil, err
	}

	return s, nil
}

func (tbl *TblTotal) GetAll() ([]*StatsTotal, error) {
	rows, err := tbl.db.Query("select * from total")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*StatsTotal
	for rows.Next() {
		s := &StatsTotal{}

		if err := rows.Scan(
			&s.UserID,
			&s.Classic,
			&s.Quote,
			&s.Ability,
			&s.AbilityCheck,
			&s.Emoji,
			&s.Splash,
			&s.SplashCheck,
			&s.DaysPlayed,
			&s.Elo,
		); err != nil {
			return nil, err
		}

		stats = append(stats, s)
	}

	return stats, nil
}

// Updates the cumulative stats for the user with UserID 'id'.
// Calculations are not performed on the database side, i.e. the database value
// is overwritten with the values in 't'. As such, updates to the current stats
// need to be handled on the application side.
func (tbl *TblTotal) Update(id string, t *StatsTotal) error {
	stmt := `
	insert into
		total
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
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
		t.DaysPlayed,
		t.Elo,
	); err != nil {
		log.Error("Failed to execute statement", "stmt", strings.ReplaceAll(stmt, "\t", "  "), "entity", t, "err", err)
		return err
	}

	return nil
}

func (tbl *TblTotal) Delete(id string) error {
	stmt := `delete from total where user_id = ?`

	if _, err := tbl.db.Exec(stmt, id); err != nil {
		log.Error("Failed to execute statement", "stmt")
		return err
	}

	return nil
}

func (tbl *TblTotal) DeleteAll() error {
	stmt := `delete from total`

	if _, err := tbl.db.Exec(stmt); err != nil {
		log.Error("Failed to execute statement", "stmt")
		return err
	}

	return nil
}
