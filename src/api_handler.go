package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"firebase.google.com/go/v4/messaging"
	"fmt"
	"github.com/appleboy/go-fcm"
	"github.com/iris-contrib/schema"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

func HandleApi(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	if err := r.ParseForm(); err != nil {
		WriteHttpError(w, r, err, http.StatusBadRequest)
		return
	}

	var notification Notification
	if err := schema.NewDecoder().Decode(&notification, r.Form); err != nil {
		WriteHttpError(w, r, err, http.StatusBadRequest)
		return
	}

	if err := notification.Validate(); err != nil {
		WriteHttpError(w, r, err, http.StatusBadRequest)
		return
	}

	notification.Credentials = Hash(notification.Credentials)

	// connect to db
	db, err := GetDB()
	if err != nil {
		WriteHttpError(w, r, err, http.StatusInternalServerError)
		return
	}

	var user User
	err = db.Table(UserTable).Get("credentials", notification.Credentials).Index("credentials-index").One(&user)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	// increase users notification count
	err = db.Table(UserTable).
		Update("device_uuid", user.UUID).
		SetExpr("notification_cnt = notification_cnt + ?", 1).
		Run()
	if err != nil {
		WriteHttpError(w, r, err, http.StatusInternalServerError)
		return
	}

	notification.Init()
	notificationMsgBytes, err := json.Marshal([]Notification{notification})
	if err != nil {
		WriteHttpError(w, r, err, http.StatusInternalServerError)
		return
	}

	if len(user.FirebaseToken) > 0 {
		credentialsJsonB64 := os.Getenv("FIREBASE_CREDENTIALS_JSON_B64")
		credentialsJson, b64Err := base64.StdEncoding.DecodeString(credentialsJsonB64)
		if b64Err != nil {
			logrus.Errorf("Problem decoding firebase message: %s", b64Err.Error())
			return
		}

		firebaseClient, err := fcm.NewClient(ctx, fcm.WithCredentialsJSON(credentialsJson))
		if err == nil && firebaseClient != nil {
			_, err := firebaseClient.Send(ctx, &messaging.Message{
				Token: user.FirebaseToken,
				Notification: &messaging.Notification{
					Title: notification.Title,
					Body:  notification.Message,
				},
			})
			if err != nil {
				logrus.Errorf("Problem sending firebase message: %s", err.Error())
			}
		} else if err != nil {
			logrus.Errorf("Problem setting up FB client: %s", err.Error())
		}
	}

	if len(user.ConnectionID) > 0 {
		err = SendWsMessage(user.ConnectionID, notificationMsgBytes)
	} else {
		err = errors.New("user has no ConnectionID")
	}
	if err != nil {
		var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
		if err := notification.Store(db, encryptionKey); err != nil {
			WriteHttpError(w, r, fmt.Errorf("%s %v", err.Error(), notification), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
