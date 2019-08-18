package main

import (
	"encoding/json"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

var VALIDCODES = map[string]int{
	"VALID":         200,
	"RESET_KEY":     401,
	"NO_UUID":       402,
	"INVALID_LOGIN": 403,
}

func (s *server) WSHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 400)
		return
	}

	if r.Header.Get("Sec-Key") != SERVERKEY {
		log.Println("Invalid sec-Key")
		http.Error(w, "Invalid key", 400)
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
	if !IsValidUUID(r.Header.Get("Uuid")) {
		http.Error(w, "Invalid UUID", 400)
		return
	} else if !IsValidVersion(r.Header.Get("Version")) {
		http.Error(w, "Invalid Version", 400)
		return
	} else if !IsValidCredentials(r.Header.Get("Credentials")) {
		http.Error(w, "Invalid Credentials", 400)
	}

	var code = 0
	UUIDUser := FetchCredentialsOfUUID(s.db, u.UUID)
	if len(UUIDUser.Credentials.Key) == 0 {
		if len(UUIDUser.Credentials.Value) == 0 {
			code = VALIDCODES["NO_UUID"]
		} else {
			log.Println("No key for", u.UUID)
			code = VALIDCODES["RESET_KEY"]
		}
	} else if !VerifyUser(s.db, u) {
		code = VALIDCODES["INVALID_LOGIN"]
	}
	if code != 0 {
		w.WriteHeader(code)
		return
	}

	if err := SetLastLogin(s.db, u); err != nil {
		log.Println("Could not set last login")
		http.Error(w, "Invalid key", 400)
	}

	// CONNECT TO SOCKET
	wsconn, _ := upgrader.Upgrade(w, r, nil)

	// add conn to clients
	clientsMutex.Lock()
	clients[u.Credentials.Value] = wsconn
	clientsMutex.Unlock()

	log.Println("Connected:", Hash(u.Credentials.Value))

	notifications, _ := FetchAllNotifications(s.db, u.Credentials.Value)
	if len(notifications) > 0 {
		bytes, _ := json.Marshal(notifications)
		if err := wsconn.WriteMessage(websocket.TextMessage, bytes); err != nil {
			log.Println(err.Error())
		}
	}

	// INCOMING SOCKET MESSAGES
	for {
		_, message, err := wsconn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			break
		}

		go DeleteNotifications(s.db, u.Credentials.Value, string(message))
	}

	clientsMutex.Lock()
	delete(clients, u.Credentials.Value)
	clientsMutex.Unlock()

	log.Println("Disconnected:", Hash(u.Credentials.Value))

	// close connection
	if err := Logout(s.db, u); err != nil {
		log.Println(err.Error())
	}
}

func (s *server) CredentialHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 400)
		return
	}

	if r.Header.Get("Sec-Key") != SERVERKEY {
		log.Println("Invalid key", r.Header.Get("Sec-Key"))
		http.Error(w, "Invalid form data", 400)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", 400)
		return
	}

	// store form data in struct
	PostUser := User{
		UUID: r.Form.Get("UUID"),

		// if asking for new credentials
		Credentials: Credentials{
			Value: r.Form.Get("current_credentials"),
			Key:   r.Form.Get("current_key"),
		},
	}

	if !IsValidUUID(PostUser.UUID) {
		http.Error(w, "Invalid form data", 400)
		return
	}

	creds, err := CreateUser(s.db, PostUser)
	if err != nil {
		println(err.Error())
	}

	c, err := json.Marshal(creds)
	_, err = w.Write(c)
	if err != nil {
		log.Println(err)
	}
}

func (s *server) APIHandler(w http.ResponseWriter, r *http.Request) {
	var n Notification

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", 400)
		return
	}

	if err := decoder.Decode(&n, r.Form); err != nil {
		http.Error(w, "Invalid form data", 400)
		return
	}

	if err := NotificationValidation(&n); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// increase notification count
	if err := IncreaseNotificationCnt(s.db, n.Credentials); err != nil {
		log.Println(err.Error())
	}

	// set ID
	n.ID = FetchTotalNumNotifications(s.db)

	// fetch client socket
	clientsMutex.RLock()
	socket, ok := clients[n.Credentials]
	clientsMutex.RUnlock()

	// send notification to client
	if ok {
		// set notification time
		t := time.Now()
		ts := t.Format("2006-01-02 15:04:05") // arbitrary values to set format
		n.Time = ts

		bytes, _ := json.Marshal([]Notification{n}) // pass as array
		if err := socket.WriteMessage(websocket.TextMessage, bytes); err != nil {
			log.Println(err.Error())
		} else {
			return // skip storing the notification as already sent to client
		}
	}

	if err := StoreNotification(s.db, n); err != nil {
		if err.(*mysql.MySQLError).Number != 1452 {
			// error other than the one saying that there are no such user credentials.
			log.Println(err.Error())
		}
	}
}
