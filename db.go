package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type Server struct {
	db *sql.DB
}

func DBConn(dataSourceName string) (db *sql.DB, err error) {
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
func removeCredKey(db *sql.DB, UUID string) {
	_, _ = db.Exec(`UPDATE users
	SET credential_key=''
	WHERE UUID=?`, Hash(UUID))
}
