package conn

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
	_ "github.com/lib/pq"
)

func PgConn() (db *sql.DB, err error) {
	psqlInfo := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s",
		os.Getenv("DATABASE_HOST"), os.Getenv("DATABASE_USER"),
		os.Getenv("DATABASE_PASS"), os.Getenv("DATABASE_NAME"))

	if len(os.Getenv("DATABASE_SSL_DISABLE")) > 0 {
		psqlInfo += " sslmode=disable"
	}

	db, err = sql.Open("postgres", psqlInfo)
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
		IdleTimeout: 1 * time.Minute,
		MaxRetries:  2,
		DB:          dbInt,
	})
	_, err := client.Ping().Result()
	return client, err
}
