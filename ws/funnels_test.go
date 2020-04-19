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

var funnels Funnels
var red *redis.Client

func TestMain(t *testing.M) {
	var err error
	red, err = conn.RedisConn(os.Getenv("redis"))
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
	conn, _ := Upgrader.Upgrade(w, r, nil)
	_, msg, _ := conn.ReadMessage()
	_ = conn.WriteMessage(1, msg)
}

func createWS(t *testing.T) *websocket.Conn {
	server := httptest.NewServer(http.HandlerFunc(webSocketHandler))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", wsURL, err)
	}
	_ = ws.SetReadDeadline(time.Now().Add(200 * time.Millisecond)) // add timeout
	return ws
}

func TestSendBytesToRemovedFunnel(t *testing.T) {
	funnels := Funnels{
		Clients: make(map[string]*Funnel),
		RWMutex: sync.RWMutex{},
	}

	key := "foo"
	funnel := &Funnel{
		WSConn: createWS(t),
		PubSub: red.Subscribe(key),
	}

	funnels.Add(funnel, key, nil)
	_ = funnels.Remove(funnel, key)

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

	key := "foo"
	funnel := &Funnel{
		WSConn: createWS(t),
		PubSub: red.Subscribe(key),
	}

	funnels.Add(funnel, key, nil)

	// send message over socket
	sendMsg := []byte("hello")
	_ = funnels.SendBytes(red, key, sendMsg)

	// read message over socket
	_, msg, _ := funnel.WSConn.ReadMessage()
	if string(msg) != string(sendMsg) {
		t.Errorf("Expected %v got %v", string(sendMsg), string(msg))
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

	key := "foo2"
	funnel := &Funnel{
		WSConn: createWS(t),
		PubSub: red.Subscribe(key),
	}
	funnels1.Add(funnel, key, func(e error) {
		fmt.Println(e.Error())
	})

	time.Sleep(50 * time.Millisecond) // wait for redis subscriber in go routine to initialise

	sendMsg := []byte("hello")
	err := funnels2.SendBytes(red, key, sendMsg)
	if err != nil {
		t.Errorf(err.Error())
	}

	_, msg, err := funnel.WSConn.ReadMessage()
	if err != nil {
		t.Errorf(err.Error())
	}
	if string(msg) != string(sendMsg) {
		t.Errorf("Expected %v got %v", string(sendMsg), string(msg))
	}
}
