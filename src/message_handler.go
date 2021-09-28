package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

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

		var notifications []Notification
		err = db.Table(NotificationTable).Get("credentials", user.Credentials).Index("credentials-index").All(&notifications)
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		if len(notifications) > 0 {
			// decrypt notifications
			for _, notification := range notifications {
				var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
				if err := notification.Decrypt(encryptionKey); err != nil {
					fmt.Println(err.Error())
				}
			}

			notificationsBytes, err := json.Marshal(notifications)
			if err != nil {
				return WriteError(err, http.StatusInternalServerError)
			}

			err = SendWsMessage(user.ConnectionID, notificationsBytes)
			if err != nil {
				return WriteError(err, http.StatusInternalServerError)
			}
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
		wtx.Delete(t.Delete("uuid", UUID).If("uuid = ?", UUID))
	}
	if err := wtx.Run(); err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return WriteEmptySuccess()
}
