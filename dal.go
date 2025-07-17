package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/log"
)

// Scanner is used to scan database values into struct fields.
//
// The interface MUST be implemented with pointer receivers in order to modify
// the underlying struct value.
type Scanner interface {
	// Scan returns a slice of struct fields to scan database values into.
	Scan() []any
}

type DAL struct {
	DB    *DB
	Today Repository[*DailyStats]
	Total Repository[*TotalStats]
}

// NewDAL returns a new DAL, initializing all repositories (see [Repository])
// with the provided database connection db.
func NewDAL(db *DB) *DAL {
	return &DAL{
		db,
		makeRepository[*DailyStats](db.Conn, "today"),
		makeRepository[*TotalStats](db.Conn, "total"),
	}
}

// Repository is a connection to a specific database table.
//
// Database tables are assumed to contain an "id"-column, which will be used for
// querying, updating, and deleting entries.
//
// The contained type T must implement scanner for storing and retrieving values
// from the underlying database. Stored fields MUST have a "db" struct-tag,
// which is used as the database column to read from / write to. All tagged
// values should be returned by t.Scan().
type Repository[T Scanner] struct {
	// Database connection used for requests.
	conn Tx

	// Name of the database table this repository interacts with.
	Tbl string

	// List of database column names for T inserted into SELECT-expressions.
	columns string

	// Parametrized string inserted into VALUES-expressions with the appropriate
	// number of parameters. The number of struct fields for a given Repository is
	// constant, such that this can be calculated during creation.
	values string
}

// makeRepository creates a new repository for T.
func makeRepository[T Scanner](conn Tx, tbl string) Repository[T] {
	// Determine database columns from struct tags
	rt := reflect.TypeFor[T]().Elem()
	col := make([]string, 0, rt.NumField())

	for i := range rt.NumField() {
		f := rt.Field(i)
		v, ok := f.Tag.Lookup("db")
		if ok {
			col = append(col, v)
		}
	}

	// Create parametrized column string for VALUES-expressions.
	b := bytes.Repeat([]byte{'?', ','}, len(col))
	val := ""
	if len(b) != 0 {
		val = string(b[:len(b)-1])
	}

	return Repository[T]{conn, tbl, strings.Join(col, ","), val}
}

// Instantiate concrete value for T. Direct instantiation is not possible in
// cases where T is a pointer type.
func (r *Repository[T]) getT() T {
	var t T

	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		t = reflect.New(rt.Elem()).Interface().(T)
	}

	return t
}

// WithTx creates a new [Repository], replacing the underlying connection with
// tx. This allows temporarily reusing r in a transactional context, where the
// transaction itself must be used to make requests.
func (r *Repository[T]) WithTx(tx Tx) *Repository[T] {
	return &Repository[T]{tx, r.Tbl, r.columns, r.values}
}

// Get fetches and returns the database entry with the given ID.
func (r *Repository[T]) Get(id string) (T, error) {
	log.Info("Getting entity", "tbl", r.Tbl, "id", id)
	stmt := fmt.Sprintf("select (%s) from %s where id = ?", r.columns, r.Tbl)

	t := r.getT()

	row := r.conn.QueryRow(stmt, id)
	if err := row.Scan(t.Scan()...); err != nil {
		log.Error("Get failed", "tbl", r.Tbl, "id", id, "stmt", stmt, "err", err)
		return t, err
	}

	log.Debug("Get complete", "tbl", r.Tbl, "id", id, "t", &t)
	return t, nil
}

// GetAll fetches all entries from the underlying database table.
func (r *Repository[T]) GetAll() ([]T, error) {
	log.Info("Getting all entities", "tbl", r.Tbl)
	stmt := fmt.Sprintf("select (%s) from %s", r.columns, r.Tbl)

	rows, err := r.conn.Query(stmt)
	if err != nil {
		log.Debug("Get all failed", "tbl", r.Tbl, "stmt", stmt, "err", err)
		return nil, err
	}
	defer rows.Close()

	var s []T
	for rows.Next() {
		t := r.getT()

		if err := rows.Scan(t.Scan()...); err != nil {
			log.Error("Get all scan failed", "tbl", r.Tbl, "stmt", stmt, "err", err)
			return nil, err
		}

		s = append(s, t)
	}

	log.Debug("Get all complete", "tbl", r.Tbl, "entities", len(s))
	return s, nil
}

// Create creates a new database entry with the given ID and data.
func (r *Repository[T]) Create(id string, t T) error {
	log.Info("Creating entity", "tbl", r.Tbl, "id", id, "entity", t)
	stmt := fmt.Sprintf("insert into %s values (%s)", r.Tbl, r.values)

	if _, err := r.conn.Exec(stmt, t.Scan()...); err != nil {
		log.Error("Create failed", "tbl", r.Tbl, "id", id, "entity", t, "stmt", stmt, "err", err)
		return err
	}

	log.Debug("Create complete", "tbl", r.Tbl, "id", id, "entity", t)
	return nil
}

// Update updates the database entry with the given ID. Returns an error if no
// such entry exists.
func (r *Repository[T]) Update(id string, t T) error {
	log.Info("Updating entity", "tbl", r.Tbl, "id", id, "entity", t)
	stmt := fmt.Sprintf("update %s set (%s) = (%s) where id = ?", r.Tbl, r.columns, r.values)

	res, err := r.conn.Exec(stmt, append(t.Scan(), id)...)
	if err != nil {
		log.Error("Update failed", "tbl", r.Tbl, "id", id, "entity", t, "stmt", stmt, "err", err)
		return err
	}
	if i, _ := res.RowsAffected(); i == 0 {
		log.Error("Update failed", "tbl", r.Tbl, "id", id, "entity", t, "stmt", stmt, "err", "no rows affected")
		return fmt.Errorf("no rows affected")
	}

	log.Debug("Update complete", "tbl", r.Tbl, "id", id, "entity", t)
	return nil
}

// Delete removes the database entry with the given ID.
func (r *Repository[T]) Delete(id string) error {
	log.Info("Deleting entity", "tbl", r.Tbl, "id", id)
	stmt := fmt.Sprintf("delete from %s where id = ?", r.Tbl)

	if _, err := r.conn.Exec(stmt, id); err != nil {
		log.Error("Delete failed", "tbl", r.Tbl, "id", id, "stmt", stmt, "err", err)
		return err
	}

	log.Debug("Delete complete", "tbl", r.Tbl, "id", id)
	return nil
}

// DeleteAll removes all entries from the underlying database table.
func (r *Repository[T]) DeleteAll() error {
	log.Info("Deleting all entities", "tbl", r.Tbl)
	stmt := fmt.Sprintf("delete from %s", r.Tbl)

	if _, err := r.conn.Exec(stmt); err != nil {
		log.Error("Delete all failed", "tbl", r.Tbl, "stmt", stmt, "err", err)
		return err
	}

	log.Debug("Delete all complete", "tbl", r.Tbl)
	return nil
}
