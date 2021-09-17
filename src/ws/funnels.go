package ws

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/websocket"
	. "github.com/maxisme/notifi-backend/logging"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

type Funnel struct {
	Channel string
	WSConn  *websocket.Conn
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

func (funnels *Funnels) Add(r *http.Request, red *redis.Client, funnel *Funnel) {
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

	funnels.pubSubWSListener(r, red, funnel.Channel)
}

func (funnels *Funnels) Remove(channel string, red *redis.Client) {
	red.Publish(channel, "close")
	// remove funnel from map
	funnels.Lock()
	delete(funnels.Clients, channel)
	funnels.Unlock()
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
	funnels.Lock()
	funnel, gotFunnel := funnels.Clients[channel]
	if gotFunnel {
		Log(r, log.InfoLevel, "Sent local message to: "+channel)
		err := funnel.WSConn.WriteMessage(websocket.TextMessage, msg)
		funnels.Unlock()
		return err
	}
	funnels.Unlock()

	_, err := red.Ping().Result()
	if err != nil {
		Log(r, log.FatalLevel, "Problem with redis conn in api: "+err.Error())
		return err
	}

	// send msg blindly to redis pub sub
	numSubscribers := red.Publish(channel, string(msg)).Val()
	Log(r, log.InfoLevel, "sent message to: "+channel+" with "+fmt.Sprintf("%d subscribers", numSubscribers))
	if numSubscribers != 0 {
		if numSubscribers == 1 {
			// successfully sent to a redis subscriber
			return nil
		} else {
			Log(r, log.ErrorLevel, fmt.Sprintf("There are more than one subscribers (%d) to this channel...", numSubscribers))
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

func (funnels *Funnels) pubSubWSListener(r *http.Request, red *redis.Client, channel string) {
	pubSub := red.Subscribe(channel)
	if err := pubSub.Ping(); err != nil {
		Log(r, log.FatalLevel, "Problem cofunnels_test.go:112 ntacting subscription: "+err.Error())
	}
	defer func(pubSub *redis.PubSub) {
		if err := pubSub.Unsubscribe(); err != nil {
			Log(r, log.FatalLevel, "problem unsubscribing: "+err.Error())
		}
	}(pubSub)

	for {
		redisMsg, err := pubSub.ReceiveMessage()
		if err == nil {
			Log(r, log.InfoLevel, channel+" received msg")
			if redisMsg.Payload == "close" {
				break
			}
			funnels.Lock()
			client, ok := funnels.Clients[channel]
			if ok {
				err := client.WSConn.WriteMessage(websocket.TextMessage, []byte(redisMsg.Payload))
				if err != nil {
					Log(r, log.FatalLevel, "problem sending funnel socket message though redis: "+err.Error())
					break
				}
			} else {
				Log(r, log.FatalLevel, "redis subscribed client doesn't have funnel")
				break
			}
			funnels.Unlock()
		} else {
			Log(r, log.FatalLevel, "problem receiving pub/sub message though redis: "+err.Error())
			if err := pubSub.Ping(); err != nil {
				Log(r, log.FatalLevel, "problem pinging redis: "+err.Error())
			} else {
				break
			}
		}
	}
	Log(r, log.InfoLevel, "closed redis listener for client: "+channel)

	funnels.Lock()
	client, ok := funnels.Clients[channel]
	if ok {
		if err := client.WSConn.Close(); err != nil {
			Log(r, log.FatalLevel, "problem force closing websocket: "+err.Error())
		}
	}
	funnels.Unlock()
}
