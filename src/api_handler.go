package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/appleboy/go-fcm"
	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"net/http"
	"os"
	"time"
)

const NotificationTimeLayout = "2006-01-02 15:04:05"

func HandleApi(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if r.HTTPMethod != "POST" && r.HTTPMethod != "GET" {
		return WriteError(errors.New("Method not allowed"), http.StatusBadRequest)
	}

	var notification Notification
	if err := json.Unmarshal([]byte(r.Body), &notification); err != nil {
		return WriteError(err, http.StatusBadRequest)
	}

	if err := notification.Validate(); err != nil {
		return WriteError(err, http.StatusBadRequest)
	}

	// connect to db
	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	// increase notification count
	err = IncreaseNotificationCnt(db, notification)
	if err != nil {
		return WriteSuccess()
	}

	// set time
	loc, _ := time.LoadLocation("UTC")
	notification.Time = time.Now().In(loc).Format(NotificationTimeLayout)

	notification.UUID = uuid.New().String()
	notificationMsgBytes, err := json.Marshal([]Notification{notification})
	if err != nil {
		return WriteError(err, http.StatusBadRequest)
	}

	user, err := GetItem(db, UserTable, "credentials", Hash(notification.Credentials))
	u := user.(User)
	if err != nil {
		return WriteError(err, http.StatusBadRequest)
	}

	if len(u.FirebaseToken) > 0 {
		msg := &fcm.Message{
			To: u.FirebaseToken,
			Notification: &fcm.Notification{
				Title: notification.Title,
				Body:  notification.Message,
				Sound: "default",
			},
		}
		firebaseClient, err := fcm.NewClient(os.Getenv("FIREBASE_SERVER_KEY"))
		if err != nil && firebaseClient != nil {
			_, err := firebaseClient.Send(msg)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	err = SendWsMessage(NewAPIGatewaySession(), u.ConnectionID, notificationMsgBytes)
	if err != nil {
		var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
		if err := notification.Store(db, encryptionKey); err != nil {
			return WriteError(err, http.StatusInternalServerError)
		}
	}

	return WriteSuccess()
}
