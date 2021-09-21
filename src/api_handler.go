package main

import (
	"encoding/json"
	"fmt"
	"github.com/appleboy/go-fcm"
	"github.com/iris-contrib/schema"
	"net/http"
	"os"
)

func HandleApi(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var notification Notification
	if err := schema.NewDecoder().Decode(&notification, r.Form); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := notification.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// connect to db
	db, err := GetDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var user User
	err = db.Table(UserTable).Get("credentials", Hash(notification.Credentials)).Index("credentials").One(&user)
	if err != nil {
		return
	}

	// increase notification count
	err = IncreaseNotificationCnt(db, user.UUID)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	notification.Init()
	notificationMsgBytes, err := json.Marshal([]Notification{notification})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		if err != nil && firebaseClient != nil {
			_, err := firebaseClient.Send(msg)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	err = SendWsMessage(NewAPIGatewaySession(), user.ConnectionID, notificationMsgBytes)
	if err != nil {
		var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
		if err := notification.Store(db, encryptionKey); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
