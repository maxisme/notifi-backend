package main

import (
	_ "database/sql"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"net/http"
	"os"
	"runtime"
)

func NewAPIGatewaySession() *apigatewaymanagementapi.ApiGatewayManagementApi {
	sesh := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region:   aws.String(os.Getenv("AWS_REGION")),
			Endpoint: aws.String(os.Getenv("WS_ENDPOINT")),
		},
	}))
	return apigatewaymanagementapi.New(sesh)
}

func WriteError(err error, code int) (events.APIGatewayProxyResponse, error) {
	_, file, no, _ := runtime.Caller(1)
	fmt.Printf("%s#%d: %s %d\n", file, no, err.Error(), code)
	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Body:       err.Error(),
	}, err
}

func WriteEmptySuccess() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil
}

func SendWsMessage(connectionID string, msgData []byte) error {
	connectionInput := &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionID),
		Data:         msgData,
	}

	_, err := NewAPIGatewaySession().PostToConnection(connectionInput)
	return err
}

func CloseConnection(connectionID string) error {
	connectionInput := &apigatewaymanagementapi.DeleteConnectionInput{
		ConnectionId: aws.String(connectionID),
	}

	_, err := NewAPIGatewaySession().DeleteConnection(connectionInput)
	return err
}

func WriteHttpError(w http.ResponseWriter, err error, code int) {
	_, file, no, _ := runtime.Caller(1)
	fmt.Printf("%s#%d: request error: %s %d\n", file, no, err.Error(), code)
	http.Error(w, err.Error(), http.StatusBadRequest)
}
