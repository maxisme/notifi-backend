package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"os"
)

var chiLambda *chiadapter.ChiLambda

func init() {
	r := chi.NewRouter()
	r.HandleFunc("/prod/code", HandleCode)
	r.HandleFunc("/prod/api", HandleApi)
	r.Use(middleware.Recoverer)
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
