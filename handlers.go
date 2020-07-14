package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/maxisme/notifi-backend/ws"

	"github.com/getsentry/sentry-go"
	"github.com/go-sql-driver/mysql"
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
		WriteError(w, r, http.StatusBadRequest, "Method not allowed")
		return
	}

	if r.Header.Get("Sec-Key") != s.key {
		WriteError(w, r, http.StatusForbidden, "Invalid Sec-Key")
		return
	}

	credentials := Credentials{
		Value:   r.Header.Get("Credentials"),
		UUIDKey: r.Header.Get("UUIDKey"),
	}
	user := User{
		Credentials: credentials,
		UUID:        r.Header.Get("Uuid"),
		AppVersion:  r.Header.Get("Version"),
	}

	fmt.Printf("%v", r.Header)

	// validate inputs
	if !IsValidUUID(user.UUID) {
		WriteError(w, r, http.StatusBadRequest, "Invalid UUID")
		return
	} else if !IsValidVersion(user.AppVersion) {
		WriteError(w, r, http.StatusBadRequest, "Invalid Version")
		return
	} else if !IsValidCredentials(user.Credentials.Value) {
		WriteError(w, r, http.StatusBadRequest, "Invalid Credentials")
		return
	}

	var errorCode = 0
	var DBUser User
	_ = DBUser.GetWithUUID(s.db, user.UUID)
	if len(DBUser.Credentials.UUIDKey) == 0 {
		errorCode = RequestNewUserCode
		if len(DBUser.Credentials.Value) == 0 {
			LogInfo(r, "No credentials or key for: "+user.UUID)
		} else {
			LogInfo(r, "No credential key for: "+user.UUID)
		}
	} else if !user.Verify(r, s.db) {
		errorCode = http.StatusForbidden
	}

	if errorCode != 0 {
		WriteError(w, r, errorCode, "Invalid login")
		return
	}

	if err := user.StoreLogin(s.db); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// connect to socket
	WSConn, err := ws.Upgrader.Upgrade(w, r, nil)
	Fatal(err)

	// initialise WS funnel
	funnel := &ws.Funnel{
		Key:    user.Credentials.Value,
		WSConn: WSConn,
		RDB:    s.redis,
	}
	s.funnels.Add(funnel)

	LogInfo(r, "Client Connected: "+crypt.Hash(user.Credentials.Value))

	LogError(r, funnel.Run(nil, false))

	LogError(r, s.funnels.Remove(funnel))

	LogInfo(r, "Client Disconnected: "+crypt.Hash(user.Credentials.Value))

	// close connection
	Fatal(user.CloseLogin(s.db))
}

// CredentialHandler is the handler for creating and updating Credentials
func (s *Server) CredentialHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		WriteError(w, r, http.StatusBadRequest, "Method not allowed")
		return
	}

	if r.Header.Get("Sec-Key") != s.key {
		WriteError(w, r, http.StatusForbidden, "Invalid Sec-Key")
		return
	}

	err := r.ParseForm()
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	fmt.Printf("%v", r.Form)

	// create PostUser struct
	PostUser := User{
		UUID: r.Form.Get("UUID"),

		// if asking for new Credentials
		Credentials: Credentials{
			Value:   r.Form.Get("current_credentials"),
			UUIDKey: r.Form.Get("current_UUIDKey"),
		},
	}

	if !IsValidUUID(PostUser.UUID) {
		LogInfo(r, "Invalid UUID:"+PostUser.UUID)
		WriteError(w, r, http.StatusBadRequest, "Invalid form data")
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
	if err := IncreaseNotificationCnt(s.db, notification); err != nil {
		// no such user with Credentials
		return
	}

	// set notification ID
	notification.Time = time.Now().Format(NotificationTimeLayout)
	notificationBytes, err := json.Marshal([]Notification{notification})
	Fatal(err)

	err = s.funnels.SendBytes(s.redis, notification.Credentials, notificationBytes)
	if err != nil {
		LogInfo(r, err.Error())
	}
}
