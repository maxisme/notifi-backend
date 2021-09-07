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

func (funnels *Funnels) Remove(funnel *Funnel, red *redis.Client) error {
	red.Publish(funnel.Channel, "close")
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
	numSubscribers := red.Publish(channel, string(msg)).Val()
	fmt.Println("sent: " + string(msg) + " to: " + channel + " with " + fmt.Sprintf("%d subscribers", numSubscribers))
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

func (funnels *Funnels) pubSubWSListener(r *http.Request, red *redis.Client, channel string) {
	pubSub := red.Subscribe(channel)
	if err := pubSub.Ping(); err != nil {
		Log(r, log.FatalLevel, "Problem contacting subscription: "+err.Error())
	}

	for {
		redisMsg, err := pubSub.ReceiveMessage()
		if err == nil {
			fmt.Println(channel + " received: " + redisMsg.Payload)
			if redisMsg.Payload == "close" {
				break
			}
			funnels.Lock()
			client := funnels.Clients[channel]
			err := client.WSConn.WriteMessage(websocket.TextMessage, []byte(redisMsg.Payload))
			if err != nil {
				Log(r, log.FatalLevel, "problem sending funnel socket message though redis: "+err.Error())
			}
			funnels.Unlock()
		} else {
			Log(r, log.FatalLevel, "problem receiving pub/sub message though redis: "+err.Error())
			if err := pubSub.Ping(); err != nil {
				Log(r, log.FatalLevel, "problem pinging redis: "+err.Error())
				time.Sleep(1 * time.Second)
			} else {
				break
			}
		}
	}
	if err := pubSub.Unsubscribe(channel); err != nil {
		Log(r, log.FatalLevel, "problem unsubscribing: "+err.Error())
	}
	fmt.Println("closed redis listener for client: " + channel)
}
