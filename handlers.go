package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/maxisme/notifi-backend/ws"

	"github.com/getsentry/sentry-go"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/maxisme/notifi-backend/crypt"
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
		Fatal(err)
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	// connect to socket
	WSConn, err := upgrader.Upgrade(w, r, nil)
	Fatal(err)

	// initialise funnel
	funnel := &ws.Funnel{
		WSConn: WSConn,
		PubSub: s.redis.Subscribe(user.Credentials.Value),
	}

	s.funnels.Add(funnel, user.Credentials.Value)

	log.Printf("Client Connected %s", crypt.Hash(user.Credentials.Value))

	// send all stored notifications from db
	go func() {
		notifications, err := user.FetchNotifications(s.db)
		Fatal(err)
		if len(notifications) > 0 {
			bytes, _ := json.Marshal(notifications)
			err := WSConn.WriteMessage(websocket.TextMessage, bytes)
			Fatal(err)
		}
	}()

	// listen for socket messages until disconnected
	for {
		_, message, err := WSConn.ReadMessage()
		if err != nil {
			// TODO handle specific err
			break // disconnected from WS
		}

		go LogErr(user.DeleteNotificationsWithIDs(s.db, string(message)))
	}

	LogErr(s.funnels.Remove(funnel, user.Credentials.Value))

	log.Println("Client Disconnected: ", crypt.Hash(user.Credentials.Value))

	// close connection
	Fatal(user.CloseLogin(s.db))
}

// CredentialHandler is the http handler for creating and updating Credentials
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

		// if asking for new Credentials
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
		Fatal(err)
	} else {
		Fatal(err)
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
		// no such user with Credentials
		return
	}

	// set notification ID
	notification.ID = FetchNumNotifications(s.db)
	notification.Time = time.Now().Format(timeLayout)
	notificationBytes, err := json.Marshal([]Notification{notification})
	Fatal(err)

	err = s.funnels.SendBytes(s.redis, notification.Credentials, notificationBytes)
	if err != nil {
		LogErr(err)
		Fatal(notification.Store(s.db))
	}
}
