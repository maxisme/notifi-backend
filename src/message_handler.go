package main

import (
	"context"
	"encoding/json"
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
		err = db.Table(UserTable).Get("connection_id", r.RequestContext.ConnectionID).Index("connection_id-index").One(&user)
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		if err := SendStoredMessages(db, user.Credentials, r.RequestContext); err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		return WriteEmptySuccess()
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
