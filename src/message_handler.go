package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
	"os"
)

const MaxWSSizeKB = 32

func HandleMessage(_ context.Context, r events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	var user User
	err = db.Table(UserTable).Get("connection_id", r.RequestContext.ConnectionID).Index("connection_id-index").One(&user)
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	if r.Body == "." {
		var notifications []Notification
		err = db.Table(NotificationTable).Get("credentials", user.Credentials).Index("credentials-index").All(&notifications)
		if err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}

		if len(notifications) > 0 {
			var notificationChunks [][]Notification
			var notificationChunk []Notification

			// decrypt notifications and chunk them into MaxWSSizeKB
			for i := range notifications {
				var notification = notifications[i]
				var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
				if err := notification.Decrypt(encryptionKey); err != nil {
					return WriteError(err, http.StatusInternalServerError)
				}

				chunkSizeBytes, _ := json.Marshal(notificationChunk)
				chunkSize := len(chunkSizeBytes)

				if chunkSize >= MaxWSSizeKB*1000 {
					notificationChunks = append(notificationChunks, notificationChunk)
					notificationChunk = []Notification{}
				}
				notificationChunk = append(notificationChunk, notification)
			}
			notificationChunks = append(notificationChunks, notificationChunk)

			// send notification chunks over websocket
			for i := range notificationChunks {
				notificationsBytes, err := json.Marshal(notificationChunks[i])
				if err != nil {
					return WriteError(err, http.StatusInternalServerError)
				}

				err = SendWsMessage(user.ConnectionID, notificationsBytes)
				if err != nil {
					return WriteError(err, http.StatusInternalServerError)
				}
			}
		}

		return WriteEmptySuccess()
	}

	var uuids []string
	if err := json.Unmarshal([]byte(r.Body), &uuids); err != nil {
		return WriteError(err, http.StatusBadRequest)
	}

	wtx := db.WriteTx()
	t := db.Table(NotificationTable)
	for _, UUID := range uuids {
		wtx.Delete(t.Delete("uuid", UUID).If("'uuid' = ?", UUID).If("'credentials' = ?", user.Credentials))
	}
	_ = wtx.Run()
	return WriteEmptySuccess()
}
