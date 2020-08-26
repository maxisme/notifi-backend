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
	"github.com/joho/godotenv"
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

// ServerKeyMiddleware middleware makes sure the Sec-Key header matches the SERVER_KEY environment variable as
// well as rate limiting requests
func ServerKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Sec-Key") != os.Getenv("SERVER_KEY") {
			WriteError(w, r, http.StatusForbidden, "Invalid server key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// load .env
	_ = godotenv.Load()

	// check all envs are set
	err := RequiredEnvs([]string{"DB_HOST", "REDIS_HOST", "ENCRYPTION_KEY", "SERVER_KEY"})
	if err != nil {
		panic(err)
	}

	// connect to db
	time.Sleep(2 * time.Second)
	fmt.Println(os.Getenv("DB_HOST"))
	dbConn, err := conn.MysqlConn(os.Getenv("DB_HOST"))
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

	// tracing
	if os.Getenv("COLLECTOR_HOSTNAME") != "" {
		// start tracer
		fn, err := tracer.InitJaegerExporter("notifi", os.Getenv("COLLECTOR_HOSTNAME"))
		if err != nil {
			panic(err)
		}
		defer fn()
	}

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

	r.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {})
	fmt.Println("Running: http://127.0.0.1:8080")
	graceful.ListenAndServe(&http.Server{Addr: ":8080", Handler: r})
}
