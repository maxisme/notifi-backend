package conn

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
	_ "github.com/go-sql-driver/mysql"
)

func MysqlConn(dataSourceName string) (db *sql.DB, err error) {
	db, err = sql.Open("mysql", dataSourceName)
	if err == nil {
		err = db.Ping()
	}
	return
}

func RedisConn(addr, db string) (*redis.Client, error) {
	dbInt, err := strconv.Atoi(db)
	if err != nil {
		return nil, err
	}
	if dbInt == 0 {
		return nil, fmt.Errorf("must specify redis db > 0")
	}
	client := redis.NewClient(&redis.Options{
		Addr:        addr,
		IdleTimeout: 1 * time.Minute,
		MaxRetries:  2,
		DB:          dbInt,
	})
	_, err = client.Ping().Result()
	return client, err
}
