package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
)

func HandleDisconnect(_ context.Context, r events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	// get user UUID from connection
	var user User
	err = db.Table(UserTable).Get("connection_id", r.RequestContext.ConnectionID).Index("connection_id-index").One(&user)
	if err == nil {
		// remove users connection_id field
		err = db.Table(UserTable).
			Update("device_uuid", user.UUID).
			Remove("connection_id").
			Run()
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}
	}
	return WriteEmptySuccess()
}
