package main

import (
	"encoding/json"
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
const NotificationTimeLayout = "2006-01-02 15:04:05"

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
			LogInfo(r, "No credentials or key for: "+user.UUID)
			errorCode = NoUUIDCode
		} else {
			LogInfo(r, "No credential key for: "+user.UUID)
			errorCode = ResetKeyCode
		}
	} else if !user.Verify(r, s.db) {
		errorCode = InvalidLoginCode
	}
	if errorCode != 0 {
		WriteError(w, r, errorCode, "Invalid login")
		return
	}

	if err := user.StoreLogin(s.db); err != nil {
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	// connect to socket
	WSConn, err := ws.Upgrader.Upgrade(w, r, nil)
	Fatal(err)

	// initialise funnel
	funnel := &ws.Funnel{
		Key:    user.Credentials.Value,
		WSConn: WSConn,
		PubSub: s.redis.Subscribe(user.Credentials.Value),
	}

	s.funnels.Add(s.redis, funnel)

	LogInfo(r, "Client Connected: "+crypt.Hash(user.Credentials.Value))

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

		go LogError(r, user.DeleteNotificationsWithIDs(s.db, string(message)))
	}

	LogError(r, s.funnels.Remove(funnel))

	LogInfo(r, "Client Disconnected: "+crypt.Hash(user.Credentials.Value))

	// close connection
	Fatal(user.CloseLogin(s.db))
}

// CredentialHandler is the handler for creating and updating Credentials
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

	creds, err := PostUser.Store(r, s.db)
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
	Fatal(err)
	if err == nil {
		_, err = w.Write(c)
		Fatal(err)
	}
}

// APIHandler is the http handler for handling API calls to create notifications
func (s *Server) APIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "GET" {
		WriteError(w, r, ErrorCode, "Method not allowed")
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	var notification Notification
	if err := decoder.Decode(&notification, r.Form); err != nil {
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	if err := notification.Validate(r); err != nil {
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	// increase notification count
	if err := IncreaseNotificationCnt(s.db, notification); err != nil {
		// no such user with Credentials
		return
	}

	// set notification ID
	notification.ID = FetchNumNotifications(s.db)
	notification.Time = time.Now().Format(NotificationTimeLayout)
	notificationBytes, err := json.Marshal([]Notification{notification})
	Fatal(err)

	err = s.funnels.SendBytes(s.redis, notification.Credentials, notificationBytes)
	if err != nil {
		Fatal(notification.Store(s.db))
	}
}
