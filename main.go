package main

import (
	"net/http"
	"os"
	"time"

	"github.com/maxisme/notifi-backend/ws"

	"github.com/TV4/graceful"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/schema"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	decoder   = schema.NewDecoder()
	serverKey = os.Getenv("server_key") // has to be passed with every request
)

// set http request limiter to max 5 requests per second
var lmt = tollbooth.NewLimiter(5, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}).SetIPLookups([]string{
	"RemoteAddr", "X-Forwarded-For", "X-Real-IP",
})

// callback function
var sentryHandler *sentryhttp.Handler

func httpCallback(nextFunc func(http.ResponseWriter, *http.Request)) http.Handler {
	if sentryHandler != nil {
		return sentryHandler.Handle(tollbooth.LimitFuncHandler(lmt, nextFunc))
	}
	return tollbooth.LimitFuncHandler(lmt, nextFunc)
}

func main() {
	// check all envs are set
	err := RequiredEnvs([]string{"db", "redis", "encryption_key", "server_key"})
	if err != nil {
		panic(err)
	}

	// connect to db
	dbConn, err := dbConn(os.Getenv("db"))
	if err != nil {
		panic(err)
	}
	defer dbConn.Close()

	// connect to redis
	redisConn, err := redisConn(os.Getenv("redis"))
	if err != nil {
		panic(err)
	}
	defer redisConn.Close()

	s := Server{
		db:      dbConn,
		redis:   redisConn,
		funnels: &ws.Funnels{Clients: make(map[credentials]*ws.Funnel)},
	}

	// init sentry
	sentryDsn := os.Getenv("sentry_dsn")
	if sentryDsn != "" {
		if err := sentry.Init(sentry.ClientOptions{Dsn: sentryDsn}); err != nil {
			panic(err.Error())
		}
		sentryHandler = sentryhttp.New(sentryhttp.Options{})
	}

	// HANDLERS
	mux := http.NewServeMux()
	mux.Handle("/ws", httpCallback(s.WSHandler))
	mux.Handle("/code", httpCallback(s.CredentialHandler))
	mux.Handle("/api", httpCallback(s.APIHandler))
	graceful.ListenAndServe(&http.Server{Addr: ":8080", Handler: mux})
}
