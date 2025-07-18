package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

// Tx represents a transaction used to access a database.
//
// A transaction may be an actual transaction, or simply a plain database
// connection. In the latter case, the "transaction" is valid for the entirety
// of the database's lifetime.
type Tx interface {
	// Exec executes a query without returning any rows (see [sql.DB.Exec]).
	Exec(query string, args ...any) (sql.Result, error)

	// Query executes a query that returns rows (see [sql.DB.Query]).
	Query(query string, args ...any) (*sql.Rows, error)

	// QueryRow executes a query that is expected to return at most one row (see
	// [sql.DB.QueryRow]).
	QueryRow(query string, args ...any) *sql.Row
}

type DB struct {
	Conn *sql.DB
}

// NewDB creates a new database connection, opening a database from a file
// called fname.
func NewDB(fname string) (*DB, error) {
	conn, err := sql.Open("sqlite3", fname)
	if err != nil {
		return nil, fmt.Errorf("open failed: %v", err)
	}

	db := &DB{conn}
	if err := db.init(); err != nil {
		return nil, fmt.Errorf("initialization failed: %v", err)
	}

	return db, nil
}

// Close closes the underlying database handle.
func (db *DB) Close() {
	db.Conn.Close()
}

// init configures and bootstraps the underlying database.
func (db *DB) init() error {
	log.Info("Executing database schema")
	if _, err := db.Conn.Exec(schema); err != nil {
		log.Error(
			"Failed to execute database schema",
			"stmt", strings.ReplaceAll(schema, "\t", "  "),
			"err", err,
		)
		return err
	}

	return nil
}

// Transaction wraps and executes fn inside of a database transaction. The
// executed function receives the transaction handle as an argument. If it
// returns an error, the transaction is rolled back and the error propagated.
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
