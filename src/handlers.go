package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"

	"github.com/maxisme/notifi-backend/ws"

	"github.com/google/uuid"
	"github.com/maxisme/notifi-backend/crypt"
)

// custom error codes
const (
	RequestNewUserCode = 551
)

// layout for times Format()
const NotificationTimeLayout = "2006-01-02 15:04:05"

// WSHandler is the http handler for web socket connections
func (s *Server) WSHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		print(r.Method)
		WriteError(w, r, http.StatusBadRequest, "Method not allowed "+r.Method)
		return
	}

	credentials := Credentials{
		Value: r.Header.Get("Credentials"),
		Key:   r.Header.Get("Key"),
	}
	user := User{
		Credentials: credentials,
		UUID:        r.Header.Get("Uuid"),
		AppVersion:  r.Header.Get("Version"),
	}

	// validate inputs
	if !IsValidUUID(user.UUID) {
		WriteError(w, r, http.StatusUnauthorized, "Invalid UUID")
		return
	} else if !IsValidVersion(user.AppVersion) {
		WriteError(w, r, http.StatusUnauthorized, fmt.Sprintf("Invalid Version %v", user.AppVersion))
		return
	} else if !IsValidCredentials(user.Credentials.Value) {
		WriteError(w, r, http.StatusUnauthorized, "Invalid Credentials")
		return
	}

	var errorCode = 0
	var DBUser User
	_ = DBUser.GetWithUUID(r, s.db, user.UUID)
	if len(DBUser.Credentials.Key) == 0 {
		errorCode = RequestNewUserCode
		if len(DBUser.Credentials.Value) == 0 {
			Log(r, log.InfoLevel, "No credentials or key for: "+user.UUID)
		} else {
			Log(r, log.InfoLevel, "No credential key for: "+user.UUID)
		}
	} else if !user.Verify(r, s.db) {
		errorCode = http.StatusForbidden
	}

	if errorCode != 0 {
		WriteError(w, r, errorCode, "Invalid login")
		return
	}

	if err := user.StoreLogin(r, s.db); err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// connect to socket
	WSConn, err := ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// initialise WS funnel
	hashedCredentials := crypt.Hash(user.Credentials.Value)
	funnel := &ws.Funnel{
		Key:    hashedCredentials,
		WSConn: WSConn,
		PubSub: s.redis.Subscribe(hashedCredentials),
	}
	s.funnels.Add(s.redis, funnel)

	Log(r, log.InfoLevel, "Client Connected: "+hashedCredentials)

	// send all stored notifications from db
	notifications, err := user.FetchNotifications(s.db)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	if len(notifications) > 0 {
		bytes, _ := json.Marshal(notifications)
		if err := WSConn.WriteMessage(websocket.TextMessage, bytes); err != nil {
			WriteError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// incoming socket messages
	for {
		_, message, err := WSConn.ReadMessage()
		if err != nil {
			break
		}
		go func() {
			if err := user.DeleteNotificationsWithIDs(r, s.db, fmt.Sprint(message)); err != nil {
				Log(r, log.WarnLevel, err)
			}
		}()
	}

	if err := s.funnels.Remove(funnel); err != nil {
		Log(r, log.WarnLevel, err)
	}

	Log(r, log.InfoLevel, "Client Disconnected: "+hashedCredentials)

	// close connection
	if err := user.CloseLogin(r, s.db); err != nil {
		Log(r, log.WarnLevel, err)
	}
}

// CredentialHandler is the handler for creating and updating Credentials
func (s *Server) CredentialHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		WriteError(w, r, http.StatusBadRequest, "Method not allowed")
		return
	}

	err := r.ParseForm()
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// create PostUser struct
	PostUser := User{
		UUID: r.Form.Get("UUID"),

		// if asking for new Credentials
		Credentials: Credentials{
			Value: r.Form.Get("current_credentials"),
			Key:   r.Form.Get("current_credential_key"),
		},
	}

	if !IsValidUUID(PostUser.UUID) {
		WriteError(w, r, http.StatusBadRequest, "Invalid UUID")
		return
	}

	creds, err := PostUser.Store(r, s.db)
	if err != nil {
		if err.Error() != "pq: duplicate key value violates unique constraint \"uuid\"" {
			Log(r, log.FatalLevel, err.Error())
		}
		WriteError(w, r, 401, err.Error()) // UUID already exists
		return
	}

	c, err := json.Marshal(creds)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	_, err = w.Write(c)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
}

// APIHandler is the http handler for handling API calls to create notifications
func (s *Server) APIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "GET" {
		WriteError(w, r, http.StatusBadRequest, "Method not allowed")
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	var notification Notification
	if err := decoder.Decode(&notification, r.Form); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := notification.Validate(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// increase notification count
	_, err := IncreaseNotificationCnt(s.db, notification)
	if err != nil {
		// no such user with Credentials
		return
	}

	// set notification
	notification.Time = time.Now().Format(NotificationTimeLayout)
	notification.UUID = uuid.New().String()
	notificationMsgBytes, err := json.Marshal([]Notification{notification})
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	err = s.funnels.SendBytes(s.redis, crypt.Hash(notification.Credentials), notificationMsgBytes)
	if err != nil {
		// store as user is not online
		var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
		if err := notification.Store(r, s.db, encryptionKey); err != nil {
			WriteError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}
}
