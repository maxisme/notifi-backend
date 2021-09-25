package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func HandleMessage(ctx context.Context, r events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	if r.Body == "." {
		db, err := GetDB()
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		var user User
		fmt.Println(r.RequestContext.ConnectionID)
		err = db.Table(UserTable).Get("connection_id", r.RequestContext.ConnectionID).Index("connection_id-index").One(&user)
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		var notifications []Notification
		err = db.Table(NotificationTable).Get("credentials", user.Credentials).All(&notifications)
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		notificationsBytes, err := json.Marshal(notifications)
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       string(notificationsBytes),
		}, nil
	}

	var uuids []string
	if err := json.Unmarshal([]byte(r.Body), &uuids); err != nil {
		return WriteError(err, http.StatusBadRequest)
	}

	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	wtx := db.WriteTx()
	t := db.Table(NotificationTable)
	for _, UUID := range uuids {
		wtx.Delete(t.Delete("uuid", UUID))
	}
	if err := wtx.Run(); err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return WriteEmptySuccess()
}
