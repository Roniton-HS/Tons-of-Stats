package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	_ "github.com/mattn/go-sqlite3"
)

type Tx interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type DB struct {
	Conn *sql.DB
}

func NewDB(fname string) (*DB, error) {
	conn, err := sql.Open("sqlite3", "tons_of_stats.sqlite")
	if err != nil {
		return nil, fmt.Errorf("open failed: %v", err)
	}

	db := &DB{conn}
	if err := db.Init(); err != nil {
		return nil, fmt.Errorf("initialization failed: %v", err)
	}

	return db, nil
}

// Closes the underlying database handle used for all connections.
func (db *DB) Close() {
	db.Conn.Close()
}

func (db *DB) Init() error {
	log.Info("Configuring database")
	var stmt string

	// Bootstrap table for daily stats
	stmt = `
	create table
		if not exists
		today (
			id            string not null primary key,
			classic       int,
			quote         int,
			ability       int,
			ability_check bool,
			emoji         int,
			splash        int,
			splash_check  bool,
			elo_change    int
		);
	`
	if _, err := db.Conn.Exec(stmt); err != nil {
		log.Error("Failed to execute statement", "stmt", strings.ReplaceAll(stmt, "\t", "  "), "err", err)
		return err
	}

	// Bootstrap table for cumulative stats
	stmt = `
	create table
		if not exists
		total (
			id            string not null primary key,
			classic       int,
			quote         int,
			ability       int,
			ability_check int,
			emoji         int,
			splash        int,
			splash_check  int,
			days_played   int,
			elo           int
		);
	`
	if _, err := db.Conn.Exec(stmt); err != nil {
		log.Error("Failed to execute statement", "stmt", strings.ReplaceAll(stmt, "\t", "  "), "err", err)
		return err
	}

	return nil
}

func (db *DB) Transaction(fn func(tx Tx) error) error {
	log.Debug("Transaction start", "fn", fn)

	tx, err := db.Conn.Begin()
	if err != nil {
		log.Debug("Transaction start failure", "fn", fn, "err", err)
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		log.Debug("Transaction internal failure", "fn", fn, "err", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Debug("Transaction commit failure", "fn", fn, "err", err)
		return err
	}

	log.Debug("Transaction complete", "fn", fn)
	return nil
}
