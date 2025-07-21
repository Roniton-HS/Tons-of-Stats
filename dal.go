package main

import (
	"tons-of-stats/db"
	"tons-of-stats/models"
)

type DAL struct {
	DB    *db.DB
	Today *db.Repository[*models.DailyStats]
	Total *db.Repository[*models.TotalStats]
}

// NewDAL returns a new DAL, initializing all repositories (see [Repository])
// with the provided database connection db.
func NewDAL(d *db.DB) *DAL {
	return &DAL{
		d,
		db.NewRepository[*models.DailyStats](d.Conn, "today"),
		db.NewRepository[*models.TotalStats](d.Conn, "total"),
	}
}
