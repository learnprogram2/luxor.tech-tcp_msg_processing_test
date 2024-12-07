package rds_db

import (
	"database/sql"
	"sync"

	_ "github.com/lib/pq"
)

var db *sql.DB
var once sync.Once

func GetDb() *sql.DB {
	once.Do(func() {
		connStr := "postgres://rds_db_admin:password@localhost:5432/postgres?sslmode=disable"
		// Open the database connection
		var err error
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			panic(err)
		}
	})
	return db
}
