package ws

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-redis/redis/v7"
	"github.com/gorilla/websocket"
)

type Funnel struct {
	Key    string
	WSConn *websocket.Conn
	PubSub *redis.PubSub
}

type Funnels struct {
	Clients map[string]*Funnel
	sync.RWMutex
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (funnels *Funnels) Add(funnel *Funnel) {
	funnels.Lock()
	funnels.Clients[funnel.Key] = funnel
	funnels.Unlock()

	// start redis subscriber listener
	go funnel.pubSubWSListener()
}

func (funnels *Funnels) Remove(funnel *Funnel) error {
	// remove client from redis subscription
	err := funnel.PubSub.Unsubscribe()
	if err != nil {
		return err
	}

	// remove funnel from map
	funnels.Lock()
	delete(funnels.Clients, funnel.Key)
	funnels.Unlock()

	return nil
}

func (funnels *Funnels) Send(red *redis.Client, key string, msg interface{}) error {
	// json encode msg
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return funnels.SendBytes(red, key, bytes)
}

func (funnels *Funnels) SendBytes(red *redis.Client, key string, msg []byte) error {
	// check if the websocket connection to this client is on this Funnels (machine)
	funnels.RLock()
	funnel, gotFunnel := funnels.Clients[key]
	funnels.RUnlock()

	if gotFunnel {
		return funnel.WSConn.WriteMessage(websocket.TextMessage, msg)
	}

	// send msg blindly to redis pub sub
	numSubscribers := red.Publish(key, string(msg))
	if numSubscribers.Val() != 0 {
		// successfully sent to a redis subscriber
		return nil
	}
	return fmt.Errorf("no socket or redis subscribers for %s", key)
}

func (funnel *Funnel) pubSubWSListener() {
	for {
		redisMsg, err := funnel.PubSub.ReceiveMessage()
		if err != nil {
			// TODO catch specific err
			break
		}
		err = funnel.WSConn.WriteMessage(websocket.TextMessage, []byte(redisMsg.Payload))
		if err != nil {
			fmt.Println("Problem sending funnel socket message though redis: " + err.Error())
		}
	}
}
