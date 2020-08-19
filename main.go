package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/maxisme/notifi-backend/conn"

	"github.com/maxisme/notifi-backend/ws"

	"github.com/TV4/graceful"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/didip/tollbooth_chi"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi"
	"github.com/gorilla/schema"
	"github.com/joho/godotenv"
)

// Server is used for database pooling - sharing the db connection to the web handlers.
type Server struct {
	db        *sql.DB
	redis     *redis.Client
	funnels   *ws.Funnels
	serverkey string
}

var (
	decoder = schema.NewDecoder()
)

const maxRequestsPerSecond = 5

func main() {
	rand.Seed(time.Now().UnixNano())

	// load .env
	_ = godotenv.Load()

	// check all envs are set
	err := RequiredEnvs([]string{"DB_SOURCE", "REDIS_HOST", "ENCRYPTION_KEY", "SERVER_KEY"})
	if err != nil {
		panic(err)
	}

	// connect to db
	dbConn, err := conn.MysqlConn(os.Getenv("DB_SOURCE"))
	if err != nil {
		panic(err)
	}
	defer dbConn.Close()

	// connect to redis
	redisConn, err := conn.RedisConn(os.Getenv("REDIS_HOST"))
	if err != nil {
		panic(err)
	}
	defer redisConn.Close()

	s := Server{
		db:        dbConn,
		redis:     redisConn,
		funnels:   &ws.Funnels{Clients: make(map[credentials]*ws.Funnel)},
		serverkey: os.Getenv("SERVER_KEY"),
	}

	// init sentry
	sentryDsn := os.Getenv("SENTRY_DSN")
	if sentryDsn != "" {
		if err := sentry.Init(sentry.ClientOptions{Dsn: sentryDsn}); err != nil {
			panic(err)
		}
	}
	sentryMiddleware := sentryhttp.New(sentryhttp.Options{})

	r := chi.NewRouter()

	// middleware
	var lmt = tollbooth.NewLimiter(maxRequestsPerSecond, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}).SetIPLookups([]string{
		"X-Forwarded-For", "X-Real-IP",
	})
	r.Use(tollbooth_chi.LimitHandler(lmt))
	r.Use(sentryMiddleware.Handle)
	AddLoggingMiddleWare(r)

	// HANDLERS
	r.HandleFunc("/ws", s.WSHandler)
	r.HandleFunc("/code", s.CredentialHandler)
	r.HandleFunc("/api", s.APIHandler)
	r.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {})
	fmt.Println("Running: http://127.0.0.1:8080")
	graceful.ListenAndServe(&http.Server{Addr: ":8080", Handler: r})
}
