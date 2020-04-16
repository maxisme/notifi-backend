package main

import (
	"encoding/json"
	"github.com/getsentry/sentry-go"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/maxisme/notifi-backend/crypt"
	"log"
	"net/http"
	"time"
)

// custom error codes
const (
	ErrorCode        = 400
	ResetKeyCode     = 401
	NoUUIDCode       = 402
	InvalidLoginCode = 403
)

// layout for times Format()
const timeLayout = "2006-01-02 15:04:05"

// WSHandler is the http handler for web socket connections
func (s *Server) WSHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		WriteError(w, r, ErrorCode, "Method not allowed")
		return
	}

	if r.Header.Get("Sec-Key") != serverKey {
		WriteError(w, r, ErrorCode, "Method not allowed")
		return
	}

	credentials := Credentials{
		Value: r.Header.Get("Credentials"),
		Key:   r.Header.Get("Credentialkey"),
	}
	user := User{
		Credentials: credentials,
		UUID:        r.Header.Get("Uuid"),
		AppVersion:  r.Header.Get("Version"),
	}

	// validate inputs
	if !IsValidUUID(user.UUID) {
		WriteError(w, r, ErrorCode, "Invalid UUID")
		return
	} else if !IsValidVersion(user.AppVersion) {
		WriteError(w, r, ErrorCode, "Invalid Version")
		return
	} else if !IsValidCredentials(user.Credentials.Value) {
		WriteError(w, r, ErrorCode, "Invalid Credentials")
		return
	}

	var errorCode = 0
	var DBUser User
	_ = DBUser.GetWithUUID(s.db, user.UUID)
	if len(DBUser.Credentials.Key) == 0 {
		if len(DBUser.Credentials.Value) == 0 {
			log.Println("No credentials or key for: " + user.UUID)
			errorCode = NoUUIDCode
		} else {
			log.Println("No credential key for: " + user.UUID)
			errorCode = ResetKeyCode
		}
	} else if !user.Verify(s.db) {
		errorCode = InvalidLoginCode
	}
	if errorCode != 0 {
		WriteError(w, r, errorCode, "Invalid login")
		return
	}

	if err := user.StoreLogin(s.db); err != nil {
		Handle(err)
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	// connect to socket
	WSConn, err := upgrader.Upgrade(w, r, nil)
	Handle(err)

	// create redis pubsub subscriber
	pubSub := s.redis.Subscribe(user.Credentials.Value)

	// initialise funnel
	funnel := &Funnel{WSConn: WSConn, pubSub: pubSub}

	s.funnels.addFunnel(funnel, user.Credentials.Value)

	log.Printf("Client Connected %s", crypt.Hash(user.Credentials.Value))

	// send all stored notifications from db
	go func() {
		notifications, err := user.FetchNotifications(s.db)
		Handle(err)
		if len(notifications) > 0 {
			bytes, _ := json.Marshal(notifications)
			err := WSConn.WriteMessage(websocket.TextMessage, bytes)
			Handle(err)
		}
	}()

	// listen for socket messages until disconnected
	for {
		_, message, err := WSConn.ReadMessage()
		if err != nil {
			// TODO handle specific err
			break // disconnected from WS
		}

		go user.DeleteNotificationsWithIDs(s.db, string(message))
	}

	s.funnels.removeFunnel(funnel, user.Credentials.Value)

	log.Println("Client Disconnected:", crypt.Hash(user.Credentials.Value))

	// close connection
	Handle(user.CloseLogin(s.db))
}

// CredentialHandler is the http handler for creating and updating credentials
func (s *Server) CredentialHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		WriteError(w, r, ErrorCode, "Method not allowed")
		return
	}

	if r.Header.Get("Sec-Key") != serverKey {
		WriteError(w, r, ErrorCode, "Method not allowed")
		return
	}

	err := r.ParseForm()
	if err != nil {
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	// convert form data to struct
	PostUser := User{
		UUID: r.Form.Get("UUID"),

		// if asking for new credentials
		Credentials: Credentials{
			Value: r.Form.Get("current_credentials"),
			Key:   r.Form.Get("current_key"),
		},
	}

	if !IsValidUUID(PostUser.UUID) {
		WriteError(w, r, ErrorCode, "Invalid form data")
		return
	}

	creds, err := PostUser.Store(s.db)
	if err != nil {
		mysqlErr, ok := err.(*mysql.MySQLError)
		if ok && mysqlErr.Number != 1062 {
			// log to sentry as a very big issue TODO what is 🤦‍
			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetLevel(sentry.LevelFatal)
				sentry.CaptureException(err)
			})
		}
		WriteError(w, r, 401, err.Error())
		return
	}

	c, err := json.Marshal(creds)
	if err == nil {
		_, err = w.Write(c)
		Handle(err)
	} else {
		Handle(err)
	}
}

// APIHandler is the http handler for handling API calls to create notifications
func (s *Server) APIHandler(w http.ResponseWriter, r *http.Request) {
	var notification Notification
	if r.Method != "POST" && r.Method != "GET" {
		WriteError(w, r, ErrorCode, "Method not allowed")
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	if err := decoder.Decode(&notification, r.Form); err != nil {
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	if err := notification.Validate(); err != nil {
		http.Error(w, err.Error(), ErrorCode)
		return
	}

	// increase notification count
	if err := IncreaseNotificationCnt(s.db, notification); err != nil {
		// no such user with credentials
		return
	}

	// set notification ID
	notification.ID = FetchNumNotifications(s.db)

	// check if websocket is connected locally
	s.funnels.RLock()
	funnel, gotSocket := s.funnels.clients[notification.credentials]
	s.funnels.RUnlock()

	if gotSocket {
		notificationBytes, err := json.Marshal([]Notification{notification})
		Handle(err)
		Handle(funnel.WSConn.WriteMessage(websocket.TextMessage, notificationBytes))
	} else {
		// look to see if there are any subscribers to this redis channel and thus a ws connection
		cmd := s.redis.PubSubChannels(notification.credentials)
		Handle(cmd.Err())
		channels, err := cmd.Result()
		Handle(err)
		if len(channels) != 0 { // there is a ws connection
			notification.Time = time.Now().Format(timeLayout)
			// convert notification to json and pass as array as that is what the client expects
			notificationBytes, err := json.Marshal([]Notification{notification})
			Handle(err)
			numSubscribers := s.redis.Publish(notification.credentials, string(notificationBytes))
			if numSubscribers.Val() != 0 {
				// sent to a redis subscriber
				return
			} else {
				log.Printf("Missing subscribers on channel %s!", notification.credentials)
			}
		}
	}

	Handle(notification.Store(s.db))
}
