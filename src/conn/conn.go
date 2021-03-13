package conn

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
	_ "github.com/lib/pq"
)

func getPgConString() string {
	psqlInfo := fmt.Sprintf("postgres://%s:%s@%s:5432/%s", os.Getenv("DATABASE_USER"),
		os.Getenv("DATABASE_PASS"), os.Getenv("DATABASE_HOST"), os.Getenv("DATABASE_NAME"))

	if len(os.Getenv("DATABASE_SSL_DISABLE")) > 0 {
		psqlInfo += "?sslmode=disable"
	}

	return psqlInfo
}

func RunPgMigration() error {
	m, err := migrate.New("file://migrations", getPgConString())
	if err != nil {
		return err
	}
	return m.Up()
}

func PgConn() (db *sql.DB, err error) {
	db, err = sql.Open("postgres", getPgConString())
	if err == nil {
		err = db.Ping()
	}
	return
}

func RedisConn() (*redis.Client, error) {
	dbInt := 0
	if len(os.Getenv("REDIS_DB")) > 0 {
		// convert db string to int
		var err error
		dbInt, err = strconv.Atoi(os.Getenv("REDIS_DB"))
		if err != nil {
			fmt.Printf("problem parsing redis db: %s\n", err)
			dbInt = 0
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:        os.Getenv("REDIS_HOST"),
		Password:    os.Getenv("REDIS_PASS"),
		IdleTimeout: 1 * time.Minute,
		MaxRetries:  2,
		DB:          dbInt,
	})
	_, err := client.Ping().Result()
	return client, err
}
