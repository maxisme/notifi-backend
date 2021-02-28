module github.com/maxisme/notifi-backend

go 1.14

require (
	cloud.google.com/go/firestore v1.3.0 // indirect
	firebase.google.com/go v3.13.0+incompatible
	github.com/TV4/graceful v0.3.4
	github.com/didip/tollbooth v4.0.2+incompatible
	github.com/didip/tollbooth_chi v0.0.0-20170928041846-6ab5f3083f3d
	github.com/getsentry/sentry-go v0.5.1
	github.com/go-chi/chi v4.0.3+incompatible
	github.com/go-errors/errors v1.0.1
	github.com/go-redis/redis/v7 v7.2.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/golang-migrate/migrate/v4 v4.8.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/schema v1.1.0
	github.com/gorilla/websocket v1.4.1
	github.com/joho/godotenv v1.3.0
	github.com/lib/pq v1.0.0
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.18.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/cobra v0.0.5
	go.opentelemetry.io/otel v0.10.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.10.0
	go.opentelemetry.io/otel/sdk v0.10.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/api v0.29.0
)
