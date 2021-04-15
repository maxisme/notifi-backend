package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"

	. "github.com/maxisme/notifi-backend/structs"
	"github.com/maxisme/notifi-backend/ws"

	"github.com/google/uuid"
	"github.com/maxisme/notifi-backend/crypt"

	"github.com/golang/gddo/httputil/header"
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
		PublicKey:   r.Header.Get("B64PublicKey"),
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
	} else if !IsValidB64PublicKey(user.PublicKey) {
		WriteError(w, r, http.StatusUnauthorized, "Invalid Public Key")
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
	notifications, err := user.FetchStoredNotifications(r, s.db)
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
		var uuids []string
		if err := json.Unmarshal(message, &uuids); err != nil {
			Log(r, log.WarnLevel, err)
			break
		}
		go func() {
			if err := user.DeleteNotificationsWithIDs(r, s.db, uuids, hashedCredentials); err != nil {
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
		UUID:      r.Form.Get("UUID"),
		PublicKey: r.Form.Get("public_key"),

		// if asking for new Credentials
		Credentials: Credentials{
			Value: r.Form.Get("current_credentials"),
			Key:   r.Form.Get("current_credential_key"),
		},
	}

	if !IsValidUUID(PostUser.UUID) {
		WriteError(w, r, http.StatusBadRequest, "Invalid UUID")
		return
	} else if !IsValidB64PublicKey(PostUser.PublicKey) {
		WriteError(w, r, http.StatusBadRequest, "Invalid public key")
		return
	}

	creds, err := PostUser.Store(r, s.db)
	if err != nil {
		if err.Error() != "pq: duplicate key value violates unique constraint \"uuid\"" {
			Log(r, log.FatalLevel, err.Error()) // TODO test
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

	var notification Notification
	contentType, _ := header.ParseValueAndParams(r.Header, "Content-Type")
	if contentType == "application/json" {
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		err := dec.Decode(&notification)
		if err != nil {
			WriteError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			WriteError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		if err := decoder.Decode(&notification, r.Form); err != nil {
			WriteError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}

	if err := Validate(r, notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// increase notification count
	err := IncreaseNotificationCnt(r, s.db, notification)
	if err != nil {
		// probably no such user with Credentials
		return
	}

	user, err := FetchUser(r, s.db, notification.Credentials)
	if err != nil {
		// probably no such user with Credentials
		return
	}

	// set time
	loc, _ := time.LoadLocation("UTC")
	notification.Time = time.Now().In(loc).Format(NotificationTimeLayout)

	notification.UUID = uuid.New().String()
	notificationMsgBytes, err := json.Marshal([]Notification{notification})
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	err = s.funnels.SendBytes(s.redis, crypt.Hash(notification.Credentials), notificationMsgBytes)
	if err != nil {
		// encrypt & store because user is probably not connected to ws
		if len(notification.EncryptedKey) == 0 {
			// notification is not already encrypted
			if err := notification.Encrypt(user.PublicKey); err != nil {
				WriteError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
		}

		if err := Store(r, s.db, notification); err != nil {
			WriteError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}
}

// KeyHandler returns the public key of a set of credentials
func (s *Server) KeyHandler(w http.ResponseWriter, r *http.Request) {
	credentials, ok := r.URL.Query()["credentials"]
	if !ok || len(credentials[0]) < 1 {
		WriteError(w, r, http.StatusBadRequest, "Missing credentials argument")
		return
	}

	if !IsValidCredentials(credentials[0]) {
		WriteError(w, r, http.StatusUnauthorized, "Invalid Credentials")
		return
	}

	user, err := FetchUser(r, s.db, credentials[0])
	if err != nil {
		// probably no such user with Credentials
		return
	}

	_, _ = w.Write([]byte(user.PublicKey))
}
