package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/go-redis/redis/v7"
	"github.com/gorilla/websocket"
)

type IncomingWSFunc func(messageType int, p []byte, err error) error

type FunnelKey = string
type Funnel struct {
	Key    string
	WSConn *websocket.Conn
	RDB    *redis.Client

	pubSub *redis.PubSub
	sync.RWMutex
}

type Funnels struct {
	Clients map[FunnelKey]*Funnel
	sync.RWMutex
}

type Message struct {
	key FunnelKey `json:"-"`
	Ref string    `json:"ref"`
	Msg []byte    `json:"msg"`
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const AckTimeout = 1 * time.Second
const WSMessageExpiration = 24 * 7 * time.Hour // 1 week

func (funnels *Funnels) Add(funnel *Funnel) {
	if err := funnel.SendStoredMessages(); err != nil {
		Log("error sending stored messages: %v", err)
	}

	// create pubsub
	funnel.pubSub = funnel.RDB.Subscribe(funnel.Key)

	// add funnel client
	funnels.Lock()
	funnels.Clients[funnel.Key] = funnel
	funnels.Unlock()

	// start pub sub listener
	go func() {
		Log("error with pub sub: %v", funnel.pubSubListener())
	}()
}

func (funnels *Funnels) Remove(funnel *Funnel) error {
	// remove client from redis subscription
	funnel.Lock()
	err := funnel.pubSub.Unsubscribe()
	funnel.Unlock()

	// remove funnel from map
	funnels.Lock()
	delete(funnels.Clients, funnel.Key)
	funnels.Unlock()
	funnel = nil
	return err
}

func (funnels *Funnels) SendBytes(RDB *redis.Client, key string, msg []byte) error {
	// check if the websocket connection to this client is on this Funnels (machine)
	funnels.RLock()
	funnel, gotFunnel := funnels.Clients[key]
	funnels.RUnlock()

	if gotFunnel {
		return funnel.writeMessage(key, msg)
	}

	// send msg blindly to redis pub sub
	// TODO worried about problem between subscriber and receiving message where message will be lost 4ever
	numSubscribers := RDB.Publish(key, string(msg))
	if numSubscribers.Val() != 0 {
		// successfully sent to a redis subscriber
		return nil
	}

	Log("storing")
	// there is no client connected to any of the servers so store the message
	if err := StoreMessage(RDB, key, uuid.New().String(), msg); err != nil {
		return err
	}
	return fmt.Errorf("no socket or redis subscribers for %s", key)
}

func (funnel *Funnel) pubSubListener() error {
	for {
		msg, err := funnel.pubSub.ReceiveMessage()
		if err != nil {
			return err
		}

		err = funnel.writeMessage(funnel.Key, []byte(msg.Payload))
		if err != nil {
			return err
		}
	}
}

func (funnel *Funnel) Acknowledge(message Message) {
	err := DeleteMessage(funnel.RDB, funnel.Key, message.Ref)
	if err != nil {
		Log("funnels: no matching message pending ack... Very strange.")
	}
}

func (funnel *Funnel) Run(wsFunc IncomingWSFunc, once bool) error {
	for {
		messageType, msg, err := funnel.WSConn.ReadMessage()
		if err != nil {
			return err
		}

		var message Message
		err = json.Unmarshal(msg, &message)
		if err != nil {
			Log("problem unmarshalling message: %v '%v'", err, string(msg))
		}

		go funnel.Acknowledge(message)

		if wsFunc != nil {
			if err := wsFunc(messageType, message.Msg, err); err != nil {
				return err //TODO perhaps allow for custom handler
			}
		}

		if once || messageType == websocket.CloseMessage {
			return nil
		}
	}
}

// SendStoredMessages writes all stored messages to client
func (funnel *Funnel) SendStoredMessages() error {
	messages, err := GetStoredMessages(funnel.RDB, funnel.Key)
	if err != nil {
		return err
	}

	for _, val := range messages {
		if err := funnel.writeMessage(funnel.Key, []byte(val)); err != nil {
			return err
		}
	}
	return nil
}

func createMessage(key string, msg []byte) (ref string, jsonMessage []byte, err error) {
	ref = uuid.New().String()
	jsonMessage, err = json.Marshal(Message{
		key,
		ref,
		msg,
	})
	if err != nil {
		return
	}
	return
}

func (funnel *Funnel) writeMessage(key string, msg []byte) error {
	ref, jsonMsg, err := createMessage(key, msg)
	if err != nil {
		return err
	}

	// write message over socket
	funnel.Lock()
	fmt.Printf("Wrote: %s\n", string(jsonMsg))
	err = funnel.WSConn.WriteMessage(websocket.TextMessage, jsonMsg)
	funnel.Unlock()
	if err != nil {
		// store message on failure // TODO not sure if I should do this :/
		if err := StoreMessage(funnel.RDB, funnel.Key, ref, msg); err != nil {
			Log("problem storing message: %v", err)
		}
		return err
	}

	return StoreMessage(funnel.RDB, funnel.Key, ref, msg)
}
