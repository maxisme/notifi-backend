package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
)

func HandleDisconnect(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	err = db.Table(ConnectionTable).Delete("ConnectionID", request.RequestContext.ConnectionID).Run()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return WriteEmptySuccess()
}
