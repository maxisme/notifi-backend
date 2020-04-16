package main

import (
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/websocket"
	"time"
)

func redisConn(addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        addr,
		IdleTimeout: 1 * time.Minute,
		MaxRetries:  2,
	})
	_, err := client.Ping().Result()
	return client, err
}

func PubSubWSListener(funnel *Funnel) {
	for {
		redisMsg, err := funnel.pubSub.ReceiveMessage()
		if err != nil {
			// TODO catch specific err
			break
		}
		go func() {
			err := funnel.WSConn.WriteMessage(websocket.TextMessage, []byte(redisMsg.Payload))
			if err != nil {
				// put back in db
				Handle(err)
			}
		}()
	}
}

func (funnels *Funnels) addFunnel(f *Funnel, key string) {
	funnels.Lock()
	funnels.clients[key] = f
	funnels.Unlock()

	// start redis listener
	go PubSubWSListener(f)
}

func (funnels *Funnels) removeFunnel(f *Funnel, key string) {
	funnels.Lock()
	// remove client from redis subscription
	Handle(f.pubSub.Unsubscribe())
	// remove funnel
	delete(funnels.clients, key)
	funnels.Unlock()
}
