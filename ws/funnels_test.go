package ws

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/go-redis/redis/v7"

	"github.com/maxisme/notifi-backend/conn"
)

var (
	RDB *redis.Client
)

func TestMain(t *testing.M) {
	var err error
	RDB, err = conn.RedisConn(os.Getenv("redis"), os.Getenv("redis_db"))
	if err != nil {
		fmt.Println("Make sure to run $ redis-server")
		panic(err)
	}

	code := t.Run() // RUN THE TEST

	// after individual test
	os.Exit(code)
}

func webSocketHandler(w http.ResponseWriter, r *http.Request) {
	c, _ := Upgrader.Upgrade(w, r, nil)
	messageType, msg, _ := c.ReadMessage()
	_ = c.WriteMessage(messageType, msg)
}

func CreateWS(t *testing.T) *websocket.Conn {
	server := httptest.NewServer(http.HandlerFunc(webSocketHandler))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", wsURL, err)
	}
	_ = ws.SetReadDeadline(time.Now().Add(1 * time.Second)) // add timeout
	return ws
}

func TestSendBytesWithoutAcknowledgement(t *testing.T) {
	funnels := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	key := randString(10)
	funnel := &Funnel{
		Key:    key,
		WSConn: CreateWS(t),
		RDB:    RDB,
	}

	funnels.Add(funnel)

	sendMsg := []byte(randString(100))
	err := funnels.SendBytes(RDB, key, sendMsg)
	if err != nil {
		t.Errorf("Should not have returned error!" + err.Error())
	}

	time.Sleep(AckTimeout * 2)

	_ = funnels.Remove(funnel)
	funnel2 := &Funnel{
		Key:    key,
		WSConn: CreateWS(t),
		RDB:    RDB,
	}
	funnels.Add(funnel2)

	err = funnel2.Run(func(messageType int, msg []byte, err error) error {
		if err != nil {
			return err
		}
		if string(msg) != string(sendMsg) {
			t.Errorf("Expected %v got %v", string(sendMsg), string(msg))
		}
		return nil
	}, true)

	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestSendBytesToRemovedFunnel(t *testing.T) {
	funnels := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	key := randString(10)
	funnel := Funnel{
		Key:    key,
		WSConn: CreateWS(t),
		RDB:    RDB,
	}

	funnels.Add(&funnel)
	// remove funnel
	_ = funnels.Remove(&funnel)

	sendMsg := []byte(randString(10))
	err := funnels.SendBytes(RDB, key, sendMsg)
	if err == nil {
		t.Errorf("Should have returned error!")
	}

	// recreate same funnel and get stored message
	funnel2 := Funnel{
		Key:    key,
		WSConn: CreateWS(t),
		RDB:    RDB,
	}
	funnels.Add(&funnel2)
	err = funnel2.Run(func(messageType int, msg []byte, err error) error {
		if string(msg) != string(sendMsg) {
			t.Errorf("Expected %v got %v", string(sendMsg), string(msg))
		}
		return nil
	}, true)

	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestSendBytesLocally(t *testing.T) {
	funnels := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	key := randString(10)
	funnel := &Funnel{
		Key:    key,
		WSConn: CreateWS(t),
		RDB:    RDB,
	}

	funnels.Add(funnel)
	defer funnels.Remove(funnel)

	// send message over socket
	sendMsg := []byte("hello")
	err := funnels.SendBytes(RDB, key, sendMsg)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	// read message over socket
	err = funnel.Run(func(messageType int, msg []byte, err error) error {
		if string(msg) != string(sendMsg) {
			t.Errorf("Expected %v got %v", string(sendMsg), string(msg))
		}
		return nil
	}, true)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestSendBytesThroughRedis(t *testing.T) {
	funnels1 := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	funnels2 := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	key := randString(10)
	funnel := &Funnel{
		Key:    key,
		WSConn: CreateWS(t),
		RDB:    RDB,
	}
	funnels1.Add(funnel)
	defer funnels1.Remove(funnel)

	sendMsg := []byte("hello")
	err := funnels2.SendBytes(RDB, key, sendMsg)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	err = funnel.Run(func(messageType int, msg []byte, err error) error {
		if string(msg) != string(sendMsg) {
			t.Errorf("Expected %v got %v", string(sendMsg), string(msg))
		}
		return nil
	}, true)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestFailedSendBytesThroughRedis(t *testing.T) {
	funnels1 := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	funnels2 := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	key := randString(10)
	funnel := &Funnel{
		Key:    key,
		WSConn: CreateWS(t),
		RDB:    RDB,
	}
	funnels1.Add(funnel)

	err := funnels1.Remove(funnel)
	if err != nil {
		t.Errorf(err.Error())
	}

	sendMsg := []byte("hello")
	err = funnels2.SendBytes(RDB, key, sendMsg)
	if err == nil {
		t.Errorf("Should have returned error")
	}
}
