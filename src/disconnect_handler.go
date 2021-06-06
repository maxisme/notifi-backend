package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"net/http"
)

func main() {
	lambda.Start(HandleDisconnect)
}

func HandleDisconnect(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	res, err := GetItem(db, UserTable, "ConnectionID", request.RequestContext.ConnectionID)
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}
	user := res.(User)
	user.ConnectionID = ""

	err = UpdateItem(db, UserTable, user.Credentials.Value, user)
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return WriteSuccess()
}
