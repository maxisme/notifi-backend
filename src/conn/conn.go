package conn

import (
	"database/sql"
	"time"

	"github.com/go-redis/redis/v7"
	_ "github.com/lib/pq"
)

func PgConn(dataSourceName string) (db *sql.DB, err error) {
	db, err = sql.Open("postgres", dataSourceName)
	if err == nil {
		err = db.Ping()
	}
	return
}

func RedisConn(addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        addr,
		IdleTimeout: 1 * time.Minute,
		MaxRetries:  2,
	})
	_, err := client.Ping().Result()
	return client, err
}
