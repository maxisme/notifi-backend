package main

import (
	"database/sql"
	"fmt"
	"github.com/go-chi/chi/middleware"
	"github.com/maxisme/notifi-backend/tracer"
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
)

// Server is used for database pooling - sharing the db connection to the web handlers.
type Server struct {
	db        *sql.DB
	redis     *redis.Client
	funnels   *ws.Funnels
	serverKey string
}

var (
	decoder = schema.NewDecoder()
)

const maxRequestsPerSecond = 5

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := conn.RunPgMigration(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	rand.Seed(time.Now().UnixNano())

	// check all envs are set
	err := RequiredEnvs([]string{"REDIS_HOST", "ENCRYPTION_KEY", "SERVER_KEY"})
	if err != nil {
		panic(err)
	}

	// connect to db
	dbConn, err := conn.PgConn()
	if err != nil {
		fmt.Println(os.Getenv("DB_HOST"))
		panic(err)
	}
	defer dbConn.Close()

	// connect to redis
	redisConn, err := conn.RedisConn()
	if err != nil {
		panic(err)
	}
	defer redisConn.Close()

	s := Server{
		db:        dbConn,
		redis:     redisConn,
		funnels:   &ws.Funnels{Clients: make(map[credentials]*ws.Funnel)},
		serverKey: os.Getenv("SERVER_KEY"),
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

	// HANDLERS
	r.Group(func(traceR chi.Router) {
		traceR.Use(tracer.Middleware)
		traceR.Use(middleware.RealIP)
		traceR.Use(middleware.Recoverer)
		traceR.Use(sentryMiddleware.Handle)
		traceR.Use(tollbooth_chi.LimitHandler(lmt))

		traceR.Group(func(secureR chi.Router) {
			secureR.Use(ServerKeyMiddleware)

			secureR.HandleFunc("/ws", s.WSHandler)
			secureR.HandleFunc("/code", s.CredentialHandler)
		})

		traceR.HandleFunc("/api", s.APIHandler)
	})

	r.HandleFunc("/", func(_ http.ResponseWriter, _ *http.Request) {})
	fmt.Println("Running: http://127.0.0.1:8080")
	graceful.ListenAndServe(&http.Server{Addr: ":8080", Handler: r})
}
