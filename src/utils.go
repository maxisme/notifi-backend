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
	"strings"
)

const (
	Region      = "us-east-1"
	WSStageName = "ws"
)

func NewAPIGatewaySession(endpoint string) *apigatewaymanagementapi.ApiGatewayManagementApi {
	sesh := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region:   aws.String(Region),
			Endpoint: aws.String(endpoint),
		},
	}))
	return apigatewaymanagementapi.New(sesh)
}

func WriteError(err error, code int) (events.APIGatewayProxyResponse, error) {
	_, file, no, _ := runtime.Caller(1)
	fmt.Printf("%s#%d: request error: %s %d\n", file, no, err.Error(), code)
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

	endpoint := strings.Replace(os.Getenv("WS_ENDPOINT"), "wss://", "https://", 1) + "/" + WSStageName
	_, err := NewAPIGatewaySession(endpoint).PostToConnection(connectionInput)
	return err
}
