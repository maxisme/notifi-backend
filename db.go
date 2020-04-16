package main

import (
	"database/sql"
	"github.com/go-redis/redis/v7"
	_ "github.com/go-sql-driver/mysql"
)

// Server is used for database pooling - sharing the db connection to the web handlers.
type Server struct {
	db      *sql.DB
	redis   *redis.Client
	funnels *Funnels
}

func dbConn(dataSourceName string) (db *sql.DB, err error) {
	db, err = sql.Open("mysql", dataSourceName)
	if err == nil {
		err = db.Ping()
	}
	return
}
