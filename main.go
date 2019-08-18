package main

import (
	"github.com/TV4/graceful"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/schema"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var decoder = schema.NewDecoder()
var SERVERKEY = os.Getenv("server_key")
var (
	clients      = make(map[string]*websocket.Conn)
	clientsMutex = sync.RWMutex{}
)

var sentryHandler *sentryhttp.Handler = nil
var lmt = tollbooth.NewLimiter(1, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}).SetIPLookups([]string{
	"RemoteAddr", "X-Forwarded-For", "X-Real-IP",
})

func customCallback(nextFunc func(http.ResponseWriter, *http.Request)) http.Handler {
	if sentryHandler != nil {
		return sentryHandler.Handle(tollbooth.LimitFuncHandler(lmt, nextFunc))
	}
	return tollbooth.LimitFuncHandler(lmt, nextFunc)
}

func main() {
	// connect to db
	db, err := DBConn(os.Getenv("db"))
	if err != nil {
		log.Fatal(err.Error())
	}
	defer db.Close()
	s := server{db: db}

	// SENTRY
	sentryDsn := os.Getenv("sentry_dsn")
	if sentryDsn != "" {
		if err := sentry.Init(sentry.ClientOptions{Dsn: sentryDsn}); err != nil {
			panic(err.Error())
		}
		sentryHandler = sentryhttp.New(sentryhttp.Options{})
	}

	// HANDLERS
	mux := http.NewServeMux()
	mux.Handle("/ws", customCallback(s.WSHandler))
	mux.Handle("/code", customCallback(s.CredentialHandler))
	mux.Handle("/api", customCallback(s.APIHandler))
	graceful.ListenAndServe(&http.Server{Addr: ":8080", Handler: mux})
}
