package ws

import (
	"fmt"
	"log"
	"sync"

	"github.com/go-redis/redis/v7"
	"github.com/gorilla/websocket"
)

type Funnel struct {
	WSConn *websocket.Conn
	PubSub *redis.PubSub
}

type Funnels struct {
	Clients map[string]*Funnel
	sync.RWMutex
}

func (funnel *Funnel) pubSubWSListener(errorHandler func(error)) {
	for {
		redisMsg, err := funnel.PubSub.ReceiveMessage()
		if err != nil {
			// TODO catch specific err
			break
		}
		err = funnel.WSConn.WriteMessage(websocket.TextMessage, []byte(redisMsg.Payload))
		errorHandler(err)
	}
}

func (funnels *Funnels) Add(f *Funnel, key string) {
	funnels.Lock()
	funnels.Clients[key] = f
	funnels.Unlock()

	// start redis listener
	go f.pubSubWSListener(func(e error) {
		log.Println(e.Error())
	})
}

func (funnels *Funnels) Remove(f *Funnel, key string) error {
	// remove client from redis subscription
	err := f.PubSub.Unsubscribe()
	if err != nil {
		return err
	}

	// remove funnel from map
	funnels.Lock()
	delete(funnels.Clients, key)
	funnels.Unlock()

	return nil
}

func (funnels *Funnels) SendBytes(red *redis.Client, key string, msg []byte) error {
	// check if the websocket connection to this client is on this machine
	funnels.RLock()
	funnel, gotFunnel := funnels.Clients[key]
	funnels.RUnlock()
	if gotFunnel {
		err := funnel.WSConn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return nil
		}
		return err
	}

	// look to see if there are any subscribers to this redis channel and thus an instance with a ws connection
	cmd := red.PubSubChannels(key)
	err := cmd.Err()
	if err != nil {
		return err
	}
	channels, err := cmd.Result()
	if err != nil {
		return err
	}
	if len(channels) != 0 {
		// there is a server with ws connections
		numSubscribers := red.Publish(key, string(msg))
		if numSubscribers.Val() != 0 {
			// sent to a redis subscriber
			return nil
		}
	}
	return fmt.Errorf("no redis subscribers for %s", key)
}
