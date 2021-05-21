package ws

import (
	"fmt"
	"math/rand"
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

var funnels Funnels
var red *redis.Client

const redisSleep = 100

func TestMain(t *testing.M) {
	var err error
	red, err = conn.RedisConn()
	if err != nil {
		fmt.Println("Make sure to run $ redis-server")
		panic(err)
	}
	funnels = Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	// create funnels
	//wg := sync.WaitGroup{}
	//for i := 1; i < 200; i++ {
	//	wg.Add(1)
	//	go func() {
	//		defer wg.Done()
	//		key := strconv.Itoa(i)
	//		funnels.Add(&Funnel{
	//			WSConn: nil,
	//			PubSub: red.Subscribe(key),
	//		}, key, nil)
	//	}()
	//}

	//wg.Wait()

	code := t.Run() // RUN THE TEST

	// after individual test
	os.Exit(code)
}

func webSocketHandler(w http.ResponseWriter, r *http.Request) {
	c, _ := Upgrader.Upgrade(w, r, nil)
	_, msg, _ := c.ReadMessage()
	_ = c.WriteMessage(1, msg)
}

func createWS(t *testing.T) *websocket.Conn {
	server := httptest.NewServer(http.HandlerFunc(webSocketHandler))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", wsURL, err)
	}
	_ = ws.SetReadDeadline(time.Now().Add(1 * time.Second)) // add timeout
	return ws
}

func TestSendBytesToRemovedFunnel(t *testing.T) {
	funnels := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	key := "foo"
	funnel := &Funnel{
		Channel: key,
		WSConn:  createWS(t),
		PubSub:  red.Subscribe(key),
	}

	funnels.Add(nil, funnel)
	_ = funnels.Remove(funnel)

	err := funnels.SendBytes(red, key, []byte("test"))
	if err == nil {
		t.Errorf("Should have returned error!")
	}
}

func TestSendBytesLocally(t *testing.T) {
	funnels := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	key := randStringBytes(10)
	funnel := &Funnel{
		Channel: key,
		WSConn:  createWS(t),
		PubSub:  red.Subscribe(key),
	}

	funnels.Add(nil, funnel)
	defer funnels.Remove(funnel) //nolint

	// send message over socket
	sendMsg := []byte("hello")
	_ = funnels.SendBytes(red, key, sendMsg)

	// read message over socket
	_, msg, _ := funnel.WSConn.ReadMessage()
	if fmt.Sprint(msg) != fmt.Sprint(sendMsg) {
		t.Errorf("Expected %v got %v", fmt.Sprint(sendMsg), fmt.Sprint(msg))
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

	key := randStringBytes(10)
	funnel := &Funnel{
		WSConn: createWS(t),
		PubSub: red.Subscribe(key),
	}
	funnels1.Add(nil, funnel)
	defer funnels1.Remove(funnel) // nolint

	time.Sleep(redisSleep * time.Millisecond) // wait for redis subscriber in go routine to initialise

	sendMsg := []byte("hello")
	err := funnels2.SendBytes(red, key, sendMsg)
	if err != nil {
		t.Errorf(err.Error())
	}

	_, msg, err := funnel.WSConn.ReadMessage()
	if err != nil {
		t.Errorf(err.Error())
	} else if string(msg) != string(sendMsg) {
		t.Errorf("Expected %v got %v", string(sendMsg), string(msg))
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

	key := randStringBytes(10)
	funnel := &Funnel{
		Channel: key,
		WSConn:  createWS(t),
		PubSub:  red.Subscribe(key),
	}
	funnels1.Add(nil, funnel)
	time.Sleep(redisSleep * time.Millisecond) // wait for redis subscriber in go routine to initialise

	err := funnels1.Remove(funnel)
	if err != nil {
		t.Errorf(err.Error())
	}

	time.Sleep(redisSleep * time.Millisecond) // wait for redis unsubscribe

	sendMsg := []byte("hello")
	err = funnels2.SendBytes(red, key, sendMsg)
	if err == nil {
		t.Errorf("Should have returned error")
	}
}

func TestStoredFailedSendBytesThroughRedis(t *testing.T) {
	funnels1 := Funnels{
		Clients:        make(map[string]*Funnel),
		StoreOnFailure: true,
		RWMutex:        sync.RWMutex{},
	}

	funnels2 := Funnels{
		Clients:        make(map[string]*Funnel),
		StoreOnFailure: true,
		RWMutex:        sync.RWMutex{},
	}

	key := randStringBytes(10)
	funnel := &Funnel{
		Channel: key,
		WSConn:  createWS(t),
		PubSub:  red.Subscribe(key),
	}
	funnels1.Add(red, funnel)
	time.Sleep(redisSleep * time.Millisecond) // wait for redis subscriber in go routine to initialise

	err := funnels1.Remove(funnel)
	if err != nil {
		t.Errorf(err.Error())
	}

	time.Sleep(redisSleep * time.Millisecond) // wait for redis unsubscribe

	sendMsg := []byte("hello")
	err = funnels2.SendBytes(red, key, sendMsg)
	if err == nil {
		t.Errorf("Should have returned error")
	}

	// add initial funnel back which should send all pending messages
	funnels1.Add(red, funnel)

	_, msg, err := funnel.WSConn.ReadMessage()
	if err != nil {
		t.Errorf(err.Error())
	} else if string(msg) != string(sendMsg) {
		t.Errorf("Expected %s got %s", sendMsg, msg)
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return fmt.Sprint(b)
}
