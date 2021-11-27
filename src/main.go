package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

var chiLambda *chiadapter.ChiLambda

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.HandleFunc("/code", HandleCode)
	r.HandleFunc("/", HandleApi)

	r.HandleFunc("/ws", func(writer http.ResponseWriter, req *http.Request) {
		http.Redirect(writer, req, "https://"+os.Getenv("WS_HOST"), http.StatusMovedPermanently)
	})
	chiLambda = chiadapter.New(r)
}

func main() {
	switch arg := os.Args[1]; arg {
	case "api":
		lambda.Start(APIHandler)
	case "connect":
		lambda.Start(HandleConnect)
	case "message":
		lambda.Start(HandleMessage)
	case "disconnect":
		lambda.Start(HandleDisconnect)
	default:
		panic("invalid lambda")
	}
}

func APIHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return chiLambda.ProxyWithContext(ctx, req)
}
