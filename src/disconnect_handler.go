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

	err = db.Table(UserTable).
		Update("connection_id", request.RequestContext.ConnectionID).
		SetExpr("connection_id = null").
		Run()

	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return WriteEmptySuccess()
}
