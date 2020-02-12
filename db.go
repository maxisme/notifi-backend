package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/maxisme/notifi-backend/crypt"
)

// Server is used for database pooling - sharing the db connection to the web handlers.
type Server struct {
	db *sql.DB
}

func dbConn(dataSourceName string) (db *sql.DB, err error) {
	db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		Handle(err)
		return
	}
	err = db.Ping()
	return
}

/////////////
// helpers //
/////////////
func removeUserCredKey(db *sql.DB, UUID string) {
	_, err := db.Exec(`UPDATE users
	SET credential_key = NULL
	WHERE UUID=?`, crypt.Hash(UUID))
	if err != nil {
		panic(err)
	}
}

func removeUserCreds(db *sql.DB, UUID string) {
	_, err := db.Exec(`UPDATE users
	SET credential_key = NULL, credentials = NULL
	WHERE UUID=?`, crypt.Hash(UUID))
	if err != nil {
		panic(err)
	}
}
