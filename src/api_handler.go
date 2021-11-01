package main

import (
	"encoding/json"
	"fmt"
	"github.com/appleboy/go-fcm"
	"github.com/iris-contrib/schema"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

func HandleApi(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteHttpError(w, err, http.StatusBadRequest)
		return
	}

	var notification Notification
	if err := schema.NewDecoder().Decode(&notification, r.Form); err != nil {
		WriteHttpError(w, err, http.StatusBadRequest)
		return
	}

	if err := notification.Validate(); err != nil {
		WriteHttpError(w, err, http.StatusBadRequest)
		return
	}

	notification.Credentials = Hash(notification.Credentials)

	// connect to db
	db, err := GetDB()
	if err != nil {
		WriteHttpError(w, err, http.StatusInternalServerError)
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
		WriteHttpError(w, err, http.StatusInternalServerError)
		return
	}

	notification.Init()
	notificationMsgBytes, err := json.Marshal([]Notification{notification})
	if err != nil {
		WriteHttpError(w, err, http.StatusInternalServerError)
		return
	}

	if len(user.FirebaseToken) > 0 {
		msg := &fcm.Message{
			To: user.FirebaseToken,
			Notification: &fcm.Notification{
				Title: notification.Title,
				Body:  notification.Message,
				Sound: "default",
			},
		}
		firebaseClient, err := fcm.NewClient(os.Getenv("FIREBASE_SERVER_KEY"))
		if err == nil && firebaseClient != nil {
			_, err := firebaseClient.Send(msg)
			if err != nil {
				logrus.Errorf("Problem sending firebase message: %s", err.Error())
			}
		} else if err != nil {
			logrus.Errorf("Problem setting up FB client: %s", err.Error())
		}
	}

	err = SendWsMessage(user.ConnectionID, notificationMsgBytes)
	if err != nil {
		logrus.Error(err.Error())
		var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
		if err := notification.Store(db, encryptionKey); err != nil {
			WriteHttpError(w, fmt.Errorf("%s %v", err.Error(), notification), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
