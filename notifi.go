package main

import (
	"encoding/json"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/schema"
	"github.com/gorilla/websocket"
	"gopkg.in/tylerb/graceful.v1"
	"log"
	"net/http"
	"os"
	"time"
)

///////////////
// variables //
///////////////
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var decoder = schema.NewDecoder()
var serverKey = os.Getenv("server_key")
var clients = make(map[string]*websocket.Conn)

//////////////
// handlers //
//////////////
func WSHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 400)
		return
	}

	if r.Header.Get("Sec-Key") != serverKey {
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

	db, err := DBConn(os.Getenv("db"))
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer db.Close()

	var code = 0
	UUIDUser := FetchCredentialsOfUUID(db, u.UUID)
	if len(UUIDUser.Credentials.Key) == 0 {
		if len(UUIDUser.Credentials.Value) == 0 {
			code = VALID_CODES["NO_UUID"]
		} else {
			log.Println("No key for", u.UUID)
			code = VALID_CODES["RESET_KEY"]
		}
	} else if !VerifyUser(db, u) {
		code = VALID_CODES["INVALID_LOGIN"]
	}
	if code != 0 {
		w.WriteHeader(code)
		return
	}

	if err := SetLastLogin(db, u); err != nil {
		log.Println("Could not set last login")
		http.Error(w, "Invalid key", 400)
	}

	wsconn, _ := upgrader.Upgrade(w, r, nil)
	clients[u.Credentials.Value] = wsconn // add conn to clients

	log.Println("Connected:", Hash(u.Credentials.Value))

	notifications, _ := FetchAllNotifications(db, u.Credentials.Value)
	if len(notifications) > 0 {
		bytes, _ := json.Marshal(notifications)
		if err := wsconn.WriteMessage(websocket.TextMessage, bytes); err != nil {
			log.Println(err.Error())
		}
	}

	for {
		_, message, err := wsconn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			break
		}

		if err = DeleteNotifications(db, u.Credentials.Value, string(message)); err != nil {
			log.Println(err.Error())
		}
	}

	delete(clients, u.Credentials.Value)
	log.Println("Disconnected:", Hash(u.Credentials.Value))

	// close connection
	if err := Logout(db, u); err != nil {
		log.Println(err.Error())
	}
}

var VALID_CODES = map[string]int{
	"VALID":         200,
	"RESET_KEY":     401,
	"NO_UUID":       402,
	"INVALID_LOGIN": 403,
}

func CredentialHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 400)
		return
	}

	if r.Header.Get("Sec-Key") != serverKey {
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

	db, err := DBConn(os.Getenv("db"))
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer db.Close()

	creds, err := CreateUser(db, PostUser)
	if err != nil {
		println(err.Error())
	}

	c, err := json.Marshal(creds)
	_, err = w.Write(c)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func APIHandler(w http.ResponseWriter, r *http.Request) {
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

	db, err := DBConn(os.Getenv("db"))
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer db.Close()

	// increase notification count
	if err = IncreaseNotificationCnt(db, n.Credentials); err != nil {
		log.Println(err.Error())
	}

	// set ID
	n.ID = FetchTotalNumNotifications(db)

	// send notification to client
	if val, ok := clients[n.Credentials]; ok {
		// set time
		t := time.Now()
		ts := t.Format("2006-01-02 15:04:05") // arbitrary values
		n.Time = ts

		bytes, _ := json.Marshal([]Notification{n}) // pass as array
		if err = val.WriteMessage(websocket.TextMessage, bytes); err != nil {
			log.Println(err.Error())
		} else {
			return // skip storing the notification as already sent to client
		}
	}

	if err = StoreNotification(db, n); err != nil {
		if err.(*mysql.MySQLError).Number != 1452 {
			// error other than the one implying that there are no such user credentials.
			log.Println(err.Error())
		}
	}
}

var sentryHandler *sentryhttp.Handler = nil
var lmt = tollbooth.NewLimiter(1, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}).SetIPLookups([]string{
	"RemoteAddr", "X-Forwarded-For", "X-Real-IP",
})

func customCallback(nextFunc func(http.ResponseWriter, *http.Request)) http.Handler {
	if sentryHandler != nil {
		return sentryHandler.Handle(tollbooth.LimitFuncHandler(lmt, nextFunc))
	}
	return tollbooth.LimitFuncHandler(lmt, nextFunc)
}

func main() {
	// SENTRY
	sentryDsn := os.Getenv("sentry_dsn")
	if sentryDsn != "" {
		if err := sentry.Init(sentry.ClientOptions{Dsn: sentryDsn}); err != nil {
			panic(err.Error())
		}
		sentryHandler = sentryhttp.New(sentryhttp.Options{})
	}

	// HANDLERS
	mux := http.NewServeMux()
	mux.Handle("/ws", customCallback(WSHandler))
	mux.Handle("/code", customCallback(CredentialHandler))
	mux.Handle("/api", customCallback(APIHandler))
	graceful.Run(":8080", 60*time.Second, mux)
}
