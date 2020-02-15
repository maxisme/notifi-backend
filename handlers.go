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

	c := Credentials{
		Value: r.Header.Get("Credentials"),
		Key:   r.Header.Get("Credentialkey"),
	}
	u := User{
		Credentials: c,
		UUID:        r.Header.Get("Uuid"),
		AppVersion:  r.Header.Get("Version"),
	}

	// validate inputs
	if !IsValidUUID(u.UUID) {
		WriteError(w, r, ErrorCode, "Invalid UUID")
		return
	} else if !IsValidVersion(u.AppVersion) {
		WriteError(w, r, ErrorCode, "Invalid Version")
		return
	} else if !IsValidCredentials(u.Credentials.Value) {
		WriteError(w, r, ErrorCode, "Invalid Credentials")
		return
	}

	var errorCode = 0
	var DBUser User
	_ = DBUser.GetWithUUID(s.db, u.UUID)
	if len(DBUser.Credentials.Key) == 0 {
		if len(DBUser.Credentials.Value) == 0 {
			log.Println("No credentials or key for: " + u.UUID)
			errorCode = NoUUIDCode
		} else {
			log.Println("No credential key for: " + u.UUID)
			errorCode = ResetKeyCode
		}
	} else if !u.Verify(s.db) {
		errorCode = InvalidLoginCode
	}
	if errorCode != 0 {
		WriteError(w, r, errorCode, "Invalid login")
		return
	}

	if err := u.StoreLogin(s.db); err != nil {
		Handle(err)
		WriteError(w, r, ErrorCode, err.Error())
		return
	}

	// connect to socket
	WSConn, err := upgrader.Upgrade(w, r, nil)
	Handle(err)

	// add conn to clients
	clientsWSMutex.Lock()
	clientsWS[u.Credentials.Value] = WSConn
	clientsWSMutex.Unlock()

	log.Println("Client Connected:", crypt.Hash(u.Credentials.Value))

	// send all pending notifications in db
	notifications, err := u.FetchNotifications(s.db)
	Handle(err)
	if len(notifications) > 0 {
		bytes, _ := json.Marshal(notifications)
		err := WSConn.WriteMessage(websocket.TextMessage, bytes)
		Handle(err)
	}

	// listen for socket messages until disconnected
	for {
		_, message, err := WSConn.ReadMessage()
		if err != nil {
			break
		}

		go Handle(u.DeleteNotificationsWithIDs(s.db, string(message)))
	}

	clientsWSMutex.Lock()
	delete(clientsWS, u.Credentials.Value)
	clientsWSMutex.Unlock()

	log.Println("Client Disconnected:", crypt.Hash(u.Credentials.Value))

	// close connection
	Handle(u.CloseLogin(s.db))
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
			// log to sentry as a very big issue
			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetLevel(sentry.LevelFatal)
				sentry.CaptureException(err)
			})
		}
		Handle(err)
		WriteError(w, r, 401, err.Error())
		return
	}

	c, err := json.Marshal(creds)
	Handle(err)
	_, err = w.Write(c)
	Handle(err)
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

	// fetch client socket
	clientsWSMutex.RLock()
	socket, gotSocket := clientsWS[notification.Credentials]
	clientsWSMutex.RUnlock()

	if gotSocket {
		// set notification time to now
		notification.Time = time.Now().Format(timeLayout)

		bytes, _ := json.Marshal([]Notification{notification}) // pass notification as array
		if err := socket.WriteMessage(websocket.TextMessage, bytes); err != nil {
			Handle(err)
		} else {
			return // skip storing the notification as already sent to client
		}
	}

	Handle(notification.Store(s.db))
}
