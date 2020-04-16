package main

import (
	"database/sql"
	"github.com/go-redis/redis/v7"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"time"
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

func redisConn(addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        addr,
		IdleTimeout: 1 * time.Minute,
		MaxRetries:  2,
	})
	_, err := client.Ping().Result()
	return client, err
}

func WSSRedisChannelListener(funnel *Funnel) {
	for {
		redisMsg, err := funnel.pubSub.ReceiveMessage()
		if err != nil {
			break
		}
		go func() {
			Handle(funnel.WSConn.WriteMessage(websocket.TextMessage, []byte(redisMsg.Payload)))
		}()
	}
}
