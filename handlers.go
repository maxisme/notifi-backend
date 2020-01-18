package main

import (
	"encoding/json"
	"github.com/getsentry/sentry-go"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const (
	ErrorCode        = 400
	ResetKeyCode     = 401
	NoUUIDCode       = 402
	InvalidLoginCode = 403

	TimeLayout = "2006-01-02 15:04:05"
)

func (s *Server) WSHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", ErrorCode)
		return
	}

	if r.Header.Get("Sec-Key") != SERVERKEY {
		WriteError(w, ErrorCode, "Invalid key")
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
		WriteError(w, ErrorCode, "Invalid UUID")
		return
	} else if !IsValidVersion(u.AppVersion) {
		WriteError(w, ErrorCode, "Invalid Version")
		return
	} else if !IsValidCredentials(u.Credentials.Value) {
		WriteError(w, ErrorCode, "Invalid Credentials")
		return
	}

	var errorCode = 0
	var DBUser User
	err := DBUser.GetWithUUID(s.db, u.UUID)
	Handle(err)
	if len(DBUser.Credentials.Key) == 0 {
		if len(DBUser.Credentials.Value) == 0 {
			errorCode = NoUUIDCode
		} else {
			log.Println("No key for: " + u.UUID)
			errorCode = ResetKeyCode
		}
	} else if !u.Verify(s.db) {
		errorCode = InvalidLoginCode
	}
	if errorCode != 0 {
		w.WriteHeader(errorCode)
		return
	}

	if err := u.StoreLogin(s.db); err != nil {
		Handle(err)
		WriteError(w, ErrorCode, err.Error())
		return
	}

	// CONNECT TO SOCKET
	WSConn, err := UPGRADER.Upgrade(w, r, nil)
	Handle(err)

	// add conn to clients
	WSClientsMutex.Lock()
	WSClients[u.Credentials.Value] = WSConn
	WSClientsMutex.Unlock()

	log.Println("Client Connected:", Hash(u.Credentials.Value))

	notifications, err := u.FetchNotifications(s.db)
	Handle(err)
	if len(notifications) > 0 {
		bytes, _ := json.Marshal(notifications)
		if err := WSConn.WriteMessage(websocket.TextMessage, bytes); err != nil {
			log.Println(err.Error())
		}
	}

	// INCOMING SOCKET MESSAGES
	for {
		_, message, err := WSConn.ReadMessage()
		if err != nil {
			Handle(err)
			break
		}

		go u.DeleteReceivedNotifications(s.db, string(message))
	}

	WSClientsMutex.Lock()
	delete(WSClients, u.Credentials.Value)
	WSClientsMutex.Unlock()

	log.Println("Client Disconnected:", Hash(u.Credentials.Value))

	// close connection
	Handle(u.CloseLogin(s.db))
}

func (s *Server) CredentialHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", ErrorCode)
		return
	}

	if r.Header.Get("Sec-Key") != SERVERKEY {
		http.Error(w, "Invalid form data", ErrorCode)
		return
	}

	err := r.ParseForm()
	if err != nil {
		Handle(err)
		http.Error(w, "Invalid form data", ErrorCode)
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
		http.Error(w, "Invalid form data", ErrorCode)
		return
	}

	creds, err := PostUser.Store(s.db)
	if err != nil {
		mysqlErr, ok := err.(*mysql.MySQLError)
		if ok && mysqlErr.Number != 1062 {
			// log to sentry as a very big issue
			sentry.CaptureMessage(mysqlErr.Message)
			sentry.Flush(time.Second * 5)
		}
		Handle(err)
		WriteError(w, 401, err.Error())
		return
	}

	c, err := json.Marshal(creds)
	Handle(err)
	_, err = w.Write(c)
	Handle(err)
}

func (s *Server) APIHandler(w http.ResponseWriter, r *http.Request) {
	var n Notification

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", ErrorCode)
		return
	}

	if err := Decoder.Decode(&n, r.Form); err != nil {
		http.Error(w, "Invalid form data", ErrorCode)
		return
	}

	if err := n.Validate(); err != nil {
		http.Error(w, err.Error(), ErrorCode)
		return
	}

	// increase notification count
	IncreaseNotificationCnt(s.db, n.Credentials)

	// set notification ID
	n.ID = FetchNumNotifications(s.db)

	// fetch client socket
	WSClientsMutex.RLock()
	socket, ok := WSClients[n.Credentials]
	WSClientsMutex.RUnlock()

	if ok {
		// set notification time
		n.Time = time.Now().Format(TimeLayout)

		bytes, _ := json.Marshal([]Notification{n}) // pass as array
		if err := socket.WriteMessage(websocket.TextMessage, bytes); err != nil {
			Handle(err)
		} else {
			return // skip storing the notification as already sent to client
		}
	}

	if err := n.Store(s.db); err != nil {
		Handle(err)
		mysqlErr, ok := err.(*mysql.MySQLError)
		if !ok || mysqlErr.Number != 1452 {
			// return any error other than the one inferring that there are no such user credentials - we don't want
			// to give that away
			Handle(err)
		}
	}
}
