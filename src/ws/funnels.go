package ws

import (
	"encoding/json"
	"fmt"
	. "github.com/maxisme/notifi-backend/logging"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/gorilla/websocket"
)

type Funnel struct {
	Channel string
	WSConn  *websocket.Conn
	PubSub  *redis.PubSub
}

type Funnels struct {
	Clients        map[string]*Funnel
	StoreOnFailure bool
	sync.RWMutex
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (funnels *Funnels) Add(r *http.Request, red *redis.Client, funnel *Funnel) error {
	funnels.Lock()
	funnels.Clients[funnel.Channel] = funnel
	funnels.Unlock()

	if funnels.StoreOnFailure {
		// read and write all temporary stored socket messages
		for {
			msg := red.LPop(funnel.Channel)
			if msg.Err() != nil || len(msg.Val()) == 0 {
				break
			}
			if err := funnel.WSConn.WriteMessage(websocket.TextMessage, []byte(msg.Val())); err != nil {
				Log(r, log.ErrorLevel, "problem writing pending redis messages: "+err.Error())
			}
		}
	}

	if err := funnel.PubSub.Ping(); err != nil {
		Log(r, log.FatalLevel, "Problem contacting subscription: "+err.Error())
		return err
	} else {
		// start redis subscriber listener
		go funnel.pubSubWSListener(r)
	}
	return nil
}

func (funnels *Funnels) Remove(funnel *Funnel) error {
	// remove client from redis subscription
	err := funnel.PubSub.Unsubscribe()
	if err != nil {
		return err
	}

	// remove funnel from map
	funnels.Lock()
	delete(funnels.Clients, funnel.Channel)
	funnels.Unlock()

	return nil
}

func (funnels *Funnels) Send(r *http.Request, red *redis.Client, channel string, msg interface{}) error {
	// json encode msg
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return funnels.SendBytes(r, red, channel, bytes)
}

func (funnels *Funnels) SendBytes(r *http.Request, red *redis.Client, channel string, msg []byte) error {
	// check if the websocket connection to this client is on this Funnels (machine)
	funnels.RLock()
	funnel, gotFunnel := funnels.Clients[channel]
	funnels.RUnlock()

	if gotFunnel {
		return funnel.WSConn.WriteMessage(websocket.TextMessage, msg)
	}

	_, err := red.Ping().Result()
	if err != nil {
		Log(r, log.FatalLevel, "Problem with redis conn in api: "+err.Error())
		return err
	}

	// send msg blindly to redis pub sub
	fmt.Println("sent: " + string(msg))
	numSubscribers := red.Publish(channel, string(msg)).Val()
	if numSubscribers != 0 {
		if numSubscribers == 1 {
			// successfully sent to a redis subscriber
			return nil
		} else {
			Log(r, log.WarnLevel, fmt.Sprintf("There are more than one subscribers (%d) to this channel...", numSubscribers))
		}
	}

	if funnels.StoreOnFailure {
		// store message in redis list due to being unable to send
		if err := red.LPush(channel, string(msg)).Err(); err != nil {
			return fmt.Errorf("unable to store message in redis map")
		}
	}

	return fmt.Errorf("no socket or redis subscribers for %s", channel)
}

func (funnel *Funnel) pubSubWSListener(r *http.Request) {
	fmt.Println("began redis listener for client: " + funnel.Channel)
	for {
		redisMsg, err := funnel.PubSub.ReceiveMessage()
		if err == nil {
			fmt.Println("received: " + redisMsg.Payload)
			if redisMsg.Payload == "close" {
				break
			}
			err = funnel.WSConn.WriteMessage(websocket.TextMessage, []byte(redisMsg.Payload))
			if err != nil {
				Log(r, log.FatalLevel, "problem sending funnel socket message though redis: "+err.Error())
			}
		} else {
			Log(r, log.FatalLevel, "problem receiving pub/sub message though redis: "+err.Error())
			if err := funnel.PubSub.Ping(); err != nil {
				Log(r, log.FatalLevel, "problem pinging redis: "+err.Error())
				time.Sleep(1 * time.Second)
			} else {
				break
			}
		}
	}
	fmt.Println("closed redis listener for client: " + funnel.Channel)
}
