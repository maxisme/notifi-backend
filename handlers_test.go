package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxisme/notifi-backend/crypt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var s Server

/////////////
// helpers //
/////////////
func PostRequest(url string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Sec-Key", os.Getenv("server_key"))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func GenUser() (Credentials, url.Values) {
	form := url.Values{}
	UUID, _ := uuid.NewRandom()
	form.Add("UUID", UUID.String())

	rr := PostRequest("", form, http.HandlerFunc(s.CredentialHandler))
	var creds Credentials
	err := json.Unmarshal(rr.Body.Bytes(), &creds)
	if err != nil {
		fmt.Println(rr.Body.String())
		panic(err)
	}
	return creds, form
}

func ConnectWSS(creds Credentials, form url.Values) (*httptest.Server, *http.Response, *websocket.Conn, error) {
	wsheader := http.Header{}
	wsheader.Add("Sec-Key", os.Getenv("server_key"))
	wsheader.Add("Credentials", creds.Value)
	wsheader.Add("Credentialkey", creds.Key)
	wsheader.Add("Uuid", form.Get("UUID"))
	wsheader.Add("Version", "1.0")

	return ConnectWSSHeader(wsheader)
}

func ConnectWSSHeader(wsheader http.Header) (*httptest.Server, *http.Response, *websocket.Conn, error) {
	s := httptest.NewServer(http.HandlerFunc(s.WSHandler))
	ws, res, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http"), wsheader)
	return s, res, ws, err
}

func SendNotification(credentials string, title string) *httptest.ResponseRecorder {
	nform := url.Values{}
	nform.Add("credentials", credentials)
	nform.Add("title", title)
	req, _ := http.NewRequest("GET", "/api?"+nform.Encode(), nil)
	rr := httptest.NewRecorder()
	http.HandlerFunc(s.APIHandler).ServeHTTP(rr, req)
	return rr
}

func readAndDelete(ws *websocket.Conn) error {
	// read
	_, mess, err := ws.ReadMessage()
	if err != nil {
		return err
	}
	var notifications []Notification
	err = json.Unmarshal(mess, &notifications)
	if err != nil {
		return err
	}

	// delete
	err = ws.WriteMessage(websocket.TextMessage, []byte(strconv.Itoa(notifications[0].ID)))
	return err
}

func removeUserCredKey(db *sql.DB, UUID string) {
	_, err := db.Exec(`UPDATE users
	SET credential_key = NULL
	WHERE UUID=?`, crypt.Hash(UUID))
	if err != nil {
		panic(err)
	}
}

func removeUserCreds(db *sql.DB, UUID string) {
	_, err := db.Exec(`UPDATE users
	SET credential_key = NULL, credentials = NULL
	WHERE UUID=?`, crypt.Hash(UUID))
	if err != nil {
		panic(err)
	}
}

// applied to every test
func TestMain(t *testing.M) {
	TESTDBNAME := "notifi_test"

	// make sure tests have all env variables
	err := RequiredEnvs([]string{"db", "encryption_key", "server_key"})
	if err != nil {
		panic(err)
	}

	// create database
	db, err := dbConn(os.Getenv("db") + "/?multiStatements=True")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %[1]v; 
	CREATE DATABASE %[1]v;`, TESTDBNAME))
	if err != nil {
		panic(err)
	}
	db.Close()

	// apply patches
	dbConnStr := os.Getenv("db") + "/" + TESTDBNAME
	m, err := migrate.New("file://sql/", "mysql://"+dbConnStr)
	if err != nil {
		panic(err)
	}

	// test up and down commands work
	if err := m.Force(1); err != nil {
		panic(err)
	}
	if err := m.Down(); err != nil {
		panic(err)
	}
	if err := m.Up(); err != nil {
		panic(err)
	}

	// init server db connection
	db, err = dbConn(dbConnStr)
	if err != nil {
		panic(err)
	}
	s = Server{db: db}

	code := t.Run() // RUN THE TEST

	// after individual test
	os.Exit(code)
}

////////////////////
// TEST FUNCTIONS //
////////////////////

func TestCredentials(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	// create new creds
	var creds, form = GenUser()
	if len(creds.Key) == 0 || len(creds.Value) == 0 {
		t.Errorf("Error getting new user credentials")
	}

	// try create a new user without specifying current credentials
	r := PostRequest("", form, http.HandlerFunc(s.CredentialHandler))
	var nocreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &nocreds)
	if len(nocreds.Value) != 0 || len(nocreds.Key) != 0 {
		t.Errorf("Shouldn't have been able to generate new creds for user!")
	}

	// ask for new credentials for user
	form.Add("current_credentials", creds.Value)
	form.Add("current_key", creds.Key)
	r = PostRequest("", form, http.HandlerFunc(s.CredentialHandler))
	var newcreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &newcreds)
	if len(newcreds.Value) == 0 || creds.Value == newcreds.Value {
		t.Errorf("Error fetching new credentials for user")
	}
}

func TestAddNotification(t *testing.T) {
	var creds, _ = GenUser()

	form := url.Values{}
	form.Add("credentials", creds.Value)
	form.Add("title", crypt.RandomString(10))

	// POST test
	r := PostRequest("", form, http.HandlerFunc(s.APIHandler))
	if status := r.Code; status != 200 {
		t.Errorf("handler returned wrong status code: got %v want %v", status, 200)
	}

	// GET test
	req, err := http.NewRequest("GET", "/api?"+form.Encode(), nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	rr := httptest.NewRecorder()
	http.HandlerFunc(s.APIHandler).ServeHTTP(rr, req)
	if status := rr.Code; status != 200 {
		t.Errorf("handler returned wrong status code: got %v want %v", status, 200)
	}
}

func TestAddNotificationWithoutTitle(t *testing.T) {
	var creds, _ = GenUser()

	form := url.Values{}
	form.Add("credentials", creds.Value)

	r := PostRequest("", form, http.HandlerFunc(s.APIHandler))
	expectedStatus := "You must enter a title!"
	status := strings.TrimSpace(r.Body.String())
	if status != expectedStatus {
		t.Errorf("handler returned wrong status code: got '%v' want '%v'", status, expectedStatus)
	}
}

func TestAddNotificationWithInvalidCredentials(t *testing.T) {
	form := url.Values{}
	form.Add("title", "test")
	form.Add("credentials", crypt.RandomString(credentialLen))

	r := PostRequest("", form, http.HandlerFunc(s.APIHandler))
	expectedStatus := ""
	if status := r.Body.String(); status != expectedStatus {
		t.Errorf("handler returned wrong status code: got '%v' want '%v'", status, expectedStatus)
	}
}

func TestSendNotificationToNonExistentUser(t *testing.T) {
	rr := SendNotification(crypt.RandomString(credentialLen), "foo")
	if rr.Code != 200 {
		t.Errorf("handler returned wrong status code: got '%d' want '%d'", rr.Code, 200)
	}
}

func TestWSHandler(t *testing.T) {
	creds, form := GenUser() // generate user

	wsheader := http.Header{}
	var headers = []struct {
		key   string
		value string
		out   bool
	}{
		{"", "", false},
		{"Sec-Key", os.Getenv("server_key"), false},
		{"Credentials", creds.Value, false},
		{"Credentialkey", creds.Key, false},
		{"Uuid", form.Get("UUID"), false},
		{"Version", "1.0.1", true},
	}

	for _, tt := range headers {
		wsheader.Add(tt.key, tt.value)
		server, _, ws, err := ConnectWSSHeader(wsheader)
		if err == nil != tt.out {
			println(tt.key + " " + tt.value)
			t.Errorf("got %v, wanted %v", err == nil, tt.out)
		}
		if ws != nil {
			ws.Close()
			server.Close()
		}
	}
}

func TestStoredNotificationsOnWSConnect(t *testing.T) {
	var creds, uform = GenUser() // generate user

	TITLE := crypt.RandomString(100)

	// send notification to not connected user
	SendNotification(creds.Value, TITLE)

	// connect to ws
	s, _, ws, _ := ConnectWSS(creds, uform)
	defer s.Close()
	defer ws.Close()

	// fetch stored notifications on Server that were sent when not connected
	_ = ws.SetReadDeadline(time.Now().Add(200 * time.Millisecond)) // add timeout
	_, mess, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf(err.Error())
	}
	var notifications []Notification
	_ = json.Unmarshal(mess, &notifications)

	if notifications[0].Title != TITLE {
		t.Error("Incorrect title returned!")
	}
}

func TestReceivingNotificationWSOnline(t *testing.T) {
	var creds, form = GenUser() // generate user

	// connect to ws
	s, _, ws, _ := ConnectWSS(creds, form)
	defer s.Close()
	defer ws.Close()

	// send notification over http
	TITLE := crypt.RandomString(10)
	SendNotification(creds.Value, TITLE)

	// read notification over ws
	_, mess, _ := ws.ReadMessage()
	var notifications []Notification
	_ = json.Unmarshal(mess, &notifications)

	if notifications[0].Title != TITLE {
		t.Error("Titles do not match! ? - ?", notifications[0].Title, TITLE)
	}
}

func TestWSSResponseCodes(t *testing.T) {
	var creds, f = GenUser() // generate user
	_, res, _, _ := ConnectWSS(creds, f)
	if res.StatusCode != 101 {
		t.Errorf("expected %v got %v", 101, res.StatusCode)
	}

	// remove credential_key
	removeUserCredKey(s.db, f.Get("UUID"))
	_, res, _, _ = ConnectWSS(creds, f)
	if res.StatusCode != ResetKeyCode {
		t.Errorf("expected %v got %v", ResetKeyCode, res.StatusCode)
	}
}

// if there is no credential_key in the db the client should be able to request new key for same credentials
// and receive a new credential key only
func TestRemovedCredentialKey(t *testing.T) {
	var _, f = GenUser() // generate user

	// remove credential_key
	removeUserCredKey(s.db, f.Get("UUID"))

	r := PostRequest("", f, http.HandlerFunc(s.CredentialHandler))
	var newCreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &newCreds)
	if len(newCreds.Key) == 0 || len(newCreds.Value) != 0 {
		t.Errorf("Error fetching new credentials for user %v. Expected new key", newCreds)
	}
}

// if there is no credentials and credential_key in the db the UUID user should
func TestRemovedCredentials(t *testing.T) {
	var _, f = GenUser() // generate user

	// remove credentials and credential_key
	removeUserCreds(s.db, f.Get("UUID"))

	r := PostRequest("", f, http.HandlerFunc(s.CredentialHandler))
	var newCreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &newCreds)

	// expects a new credential key to be returned only
	if len(newCreds.Key) == 0 || len(newCreds.Value) == 0 {
		t.Errorf("Error fetching new credentials for user %v. Expected new credentials and key", newCreds)
	}
}

var invalidHandlerMethods = []struct {
	handler       http.HandlerFunc
	invalidMethod string
}{
	{http.HandlerFunc(s.CredentialHandler), "GET"},
	{http.HandlerFunc(s.APIHandler), "PUT"},
	{http.HandlerFunc(s.WSHandler), "POST"},
}

// request handlers with incorrect methods
func TestInvalidHandlerMethods(t *testing.T) {
	for i, tt := range invalidHandlerMethods {
		t.Run(string(i), func(t *testing.T) {
			req, _ := http.NewRequest(tt.invalidMethod, "", nil)

			rr := httptest.NewRecorder()
			tt.handler.ServeHTTP(rr, req)
			if rr.Code != ErrorCode {
				t.Errorf("Should have responded with error code %d not %d", ErrorCode, rr.Code)
			}
		})
	}
}

var secKeyHandlers = []struct {
	handler http.HandlerFunc
	method  string
}{
	{http.HandlerFunc(s.CredentialHandler), "POST"},
	{http.HandlerFunc(s.WSHandler), "GET"},
}

// test handlers with correct methods but no secret server_key
func TestMissingSecKeyHandlers(t *testing.T) {
	for i, tt := range secKeyHandlers {
		t.Run(string(i), func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "", nil)
			rr := httptest.NewRecorder()
			tt.handler.ServeHTTP(rr, req)
			if rr.Code != ErrorCode {
				t.Errorf("Should have responded with error code %d not %d", ErrorCode, rr.Code)
			}
		})
	}
}

func TestDeleteNotificationsWithIncorrectIDs(t *testing.T) {
	var userCreds, form = GenUser() // generate user

	// send 3 notifications to user
	SendNotification(userCreds.Value, crypt.RandomString(10))
	SendNotification(userCreds.Value, crypt.RandomString(10))
	SendNotification(userCreds.Value, crypt.RandomString(10))

	// read notifications over ws
	sock, _, ws, _ := ConnectWSS(userCreds, form)
	defer sock.Close()
	defer ws.Close()
	_, mess, _ := ws.ReadMessage()
	var notifications []Notification
	_ = json.Unmarshal(mess, &notifications)

	// fetch ids from notifications
	var ids string
	for _, notification := range notifications {
		ids = fmt.Sprintf("%s%d,", ids, notification.ID)
	}

	u := User{Credentials: Credentials{Value: userCreds.Value}}

	err := u.DeleteNotificationsWithIDs(s.db, ids) // correct ids
	if err != nil {
		t.Errorf("Expected no error got: %v", err)
	}
}

// send notification while offline, connect to websocket to receive said notification
// tell Server to delete notification, reconnect to websocket and service should not receive a message
func TestDeleteNotification(t *testing.T) {
	var creds, uform = GenUser() // generate user

	// send notification to not connected user
	SendNotification(creds.Value, crypt.RandomString(10))

	// connect to wss
	s, _, ws, _ := ConnectWSS(creds, uform)

	// delete notification
	if err := readAndDelete(ws); err != nil {
		t.Errorf("Shouldn't have returned error: " + err.Error())
	}

	// disconnect from ws
	s.Close()
	ws.Close()

	// reconnect to ws
	_, _, ws, _ = ConnectWSS(creds, uform)

	// expect timeout on read notification
	_ = ws.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err := ws.ReadMessage()
	if err == nil {
		t.Errorf("Should have had i/o timeout and received nothing")
	}
}

func TestFullLifeSpanConcurrency(t *testing.T) {
	var numAccounts = 5
	var numNotifications = 10

	var wg sync.WaitGroup

	// generate multiple accounts
	for i := 0; i < numAccounts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// create user
			var creds, uform = GenUser()

			// send notifications
			for i := 0; i < numNotifications; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					SendNotification(creds.Value, crypt.RandomString(100))
				}()
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				// connect to web socket
				s, _, ws, err := ConnectWSS(creds, uform)
				if err != nil {
					panic(err)
				}
				defer s.Close()
				defer ws.Close()

				// read and delete all notifications
				for i := 0; i < numNotifications; i++ {
					if err := readAndDelete(ws); err != nil {
						t.Errorf("Shouldn't have returned error: " + err.Error())
					}
				}
				if err := readAndDelete(ws); err == nil {
					t.Errorf("Should have returned error as read all messages")
				}
			}()
		}()
	}
	wg.Wait()
}
