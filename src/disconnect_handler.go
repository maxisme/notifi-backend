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

	var user User
	err = db.Table(UserTable).Get("ConnectionID", request.RequestContext.ConnectionID).One(&user)
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}
	user.ConnectionID = ""

	err = UpdateItem(db, UserTable, user.Credentials, user)
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return WriteEmptySuccess()
}
