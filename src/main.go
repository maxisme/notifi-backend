package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"net/http"
	"os"
	"time"
)

var chiLambda *chiadapter.ChiLambda

func init() {
	db, err := GetDB()
	if err != nil {
		panic(err.Error())
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(httprate.Limit(
		30,
		1*time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
		httprate.WithLimitCounter(getLimitCounter(db)),
	))
	r.HandleFunc("/code", HandleCode)
	r.HandleFunc("/api", HandleApi)
	r.HandleFunc("/ws", func(writer http.ResponseWriter, req *http.Request) {
		http.Redirect(writer, req, "https://"+os.Getenv("WS_HOST"), http.StatusMovedPermanently)
	})
	chiLambda = chiadapter.New(r)
}

func main() {
	switch arg := os.Args[1]; arg {
	case "http":
		lambda.Start(HttpHandler)
	case "connect":
		lambda.Start(HandleConnect)
	case "message":
		lambda.Start(HandleMessage)
	case "disconnect":
		lambda.Start(HandleDisconnect)
	default:
		panic("missing args")
	}
}

func HttpHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return chiLambda.ProxyWithContext(ctx, req)
}
