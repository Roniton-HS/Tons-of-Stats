package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/log"
)

type Scanner interface {
	Scan() []any
}

type DAL struct {
	DB    *DB
	Today Repository[*DailyStats]
	Total Repository[*TotalStats]
}

func NewDAL(db *DB) *DAL {
	return &DAL{
		db,
		makeRepository[*DailyStats](db.Conn, "today"),
		makeRepository[*TotalStats](db.Conn, "total"),
	}
}

type Repository[T Scanner] struct {
	conn  Tx
	Table string

	// List of database column names for T. Fields on T that are saved in the
	// database must have a "db" struct-tag. All tagged fields should be returned
	// by t.Scan().
	columns string

	// Parametrized string inserted into VALUES-expressions with the appropriate
	// number of parameters. The number of struct fields for a given Repository[T]
	// is constant, such that this can be calculated during creation.
	values string
}

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

func (r *Repository[T]) WithTx(tx Tx) *Repository[T] {
	return &Repository[T]{tx, r.Table, r.columns, r.values}
}

func (r *Repository[T]) Get(id string) (T, error) {
	log.Info("Getting entity", "tbl", r.Table, "id", id)
	stmt := fmt.Sprintf("select * from %s where id = ?", r.Table)

	t := r.getT()

	row := r.conn.QueryRow(stmt, id)
	if err := row.Scan(t.Scan()...); err != nil {
		log.Error("Get failed", "tbl", r.Table, "id", id, "stmt", stmt, "err", err)
		return t, err
	}

	log.Debug("Get complete", "tbl", r.Table, "id", id, "t", &t)
	return t, nil
}

func (r *Repository[T]) GetAll() ([]T, error) {
	log.Info("Getting all entities", "tbl", r.Table)
	stmt := fmt.Sprintf("select * from %s", r.Table)

	rows, err := r.conn.Query(stmt)
	if err != nil {
		log.Debug("Get all failed", "tbl", r.Table, "stmt", stmt, "err", err)
		return nil, err
	}
	defer rows.Close()

	var s []T
	for rows.Next() {
		t := r.getT()

		if err := rows.Scan(t.Scan()...); err != nil {
			log.Error("Get all scan failed", "tbl", r.Table, "stmt", stmt, "err", err)
			return nil, err
		}

		s = append(s, t)
	}

	log.Debug("Get all complete", "tbl", r.Table, "entities", len(s))
	return s, nil
}

func (r *Repository[T]) Create(id string, t T) error {
	log.Info("Creating entity", "tbl", r.Table, "id", id, "entity", t)
	stmt := fmt.Sprintf("insert into %s values (%s)", r.Table, r.values)

	if _, err := r.conn.Exec(stmt, t.Scan()...); err != nil {
		log.Error("Create failed", "tbl", r.Table, "id", id, "entity", t, "stmt", stmt, "err", err)
		return err
	}

	log.Debug("Create complete", "tbl", r.Table, "id", id, "entity", t)
	return nil
}

func (r *Repository[T]) Update(id string, t T) error {
	log.Info("Updating entity", "tbl", r.Table, "id", id, "entity", t)
	stmt := fmt.Sprintf("update %s set (%s) = (%s) where id = ?", r.Table, r.columns, r.values)

	res, err := r.conn.Exec(stmt, append(t.Scan(), id)...)
	if err != nil {
		log.Error("Update failed", "tbl", r.Table, "id", id, "entity", t, "stmt", stmt, "err", err)
		return err
	}
	if i, _ := res.RowsAffected(); i == 0 {
		log.Error("Update failed", "tbl", r.Table, "id", id, "entity", t, "stmt", stmt, "err", "no rows affected")
		return fmt.Errorf("no rows affected")
	}

	log.Debug("Update complete", "tbl", r.Table, "id", id, "entity", t)
	return nil
}

func (r *Repository[T]) Delete(id string) error {
	log.Info("Deleting entity", "tbl", r.Table, "id", id)
	stmt := fmt.Sprintf("delete from %s where id = ?", r.Table)

	if _, err := r.conn.Exec(stmt, id); err != nil {
		log.Error("Delete failed", "tbl", r.Table, "id", id, "stmt", stmt, "err", err)
		return err
	}

	log.Debug("Delete complete", "tbl", r.Table, "id", id)
	return nil
}

func (r *Repository[T]) DeleteAll() error {
	log.Info("Deleting all entities", "tbl", r.Table)
	stmt := fmt.Sprintf("delete from %s", r.Table)

	if _, err := r.conn.Exec(stmt); err != nil {
		log.Error("Delete all failed", "tbl", r.Table, "stmt", stmt, "err", err)
		return err
	}

	log.Debug("Delete all complete", "tbl", r.Table)
	return nil
}
