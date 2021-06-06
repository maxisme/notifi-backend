package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(HandleMessage)
}

func HandleMessage(ctx context.Context, r events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	if r.Body == "." {
		db, err := GetDB()
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}
		result, err := GetItem(db, UserTable, "ConnectionID", r.RequestContext.ConnectionID)
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		user := result.(User)
		if err := SendStoredMessages(db, user.Credentials.Value, r.RequestContext.ConnectionID); err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		return WriteSuccess()
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

	return WriteSuccess()
}
