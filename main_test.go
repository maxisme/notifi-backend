package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

var s server

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
	_ = json.Unmarshal(rr.Body.Bytes(), &creds)
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

func SendNotification(credentials string, title string) {
	nform := url.Values{}
	nform.Add("credentials", credentials)
	nform.Add("title", title)
	req, _ := http.NewRequest("GET", "/api?"+nform.Encode(), nil)
	rr := httptest.NewRecorder()
	http.HandlerFunc(s.APIHandler).ServeHTTP(rr, req)
}

// applied to every test
func TestMain(m *testing.M) {
	// initialise db
	db, err := DBConn(os.Getenv("test_db_host") + "/?multiStatements=True")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	schema, _ := ioutil.ReadFile("sql/schema.sql")
	_, err = db.Exec(`DROP DATABASE IF EXISTS notifi_test; 
	CREATE DATABASE notifi_test;
	USE notifi_test; ` + string(schema))
	if err != nil {
		panic(err)
	}

	db, err = DBConn(os.Getenv("db"))
	s = server{db: db}

	code := m.Run() // RUN THE TEST

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
	form.Add("title", RandomString(10))

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
	expected_status := "You must enter a title!\n"
	if status := r.Body.String(); status != expected_status {
		t.Errorf("handler returned wrong status code: got '%v' want '%v'", status, expected_status)
	}
}

func TestAddNotificationWithInvalidCredentials(t *testing.T) {
	form := url.Values{}
	form.Add("title", RandomString(25))
	form.Add("credentials", RandomString(25))

	r := PostRequest("", form, http.HandlerFunc(s.APIHandler))
	expected_status := ""
	if status := r.Body.String(); status != expected_status {
		t.Errorf("handler returned wrong status code: got '%v' want '%v'", status, expected_status)
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

	TITLE := RandomString(100)

	// send notification to not connected user
	SendNotification(creds.Value, TITLE)

	// connect to ws
	s, _, ws, _ := ConnectWSS(creds, uform)
	defer s.Close()
	defer ws.Close()

	// fetch stored notifications on server that were sent when not connected
	_ = ws.SetReadDeadline(time.Now().Add(100 * time.Millisecond)) // add timeout
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

// send notification while offline, connect to websocket to receive said notification
// tell server to delete notification, reconnect to websocket and service should not recieve a message
func TestDeleteNotification(t *testing.T) {
	var creds, uform = GenUser() // generate user

	// send notification to not connected user
	SendNotification(creds.Value, RandomString(10))

	// connect to wss
	s, _, ws, _ := ConnectWSS(creds, uform)

	// delete notification
	_, mess, _ := ws.ReadMessage()
	var notifications []Notification
	_ = json.Unmarshal(mess, &notifications)
	_ = ws.WriteMessage(websocket.TextMessage, []byte(strconv.Itoa(notifications[0].ID)))

	// disconnect from ws
	s.Close()
	ws.Close()

	// reconnect to ws
	s, _, ws, _ = ConnectWSS(creds, uform)

	// expect timeout on read notification
	_ = ws.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err := ws.ReadMessage()
	if err == nil {
		t.Errorf("Should have had i/o timeout and received nothing")
	}
}

func TestReceivingNotificationWSOnline(t *testing.T) {
	var creds, form = GenUser() // generate user

	// connect to ws
	s, _, ws, _ := ConnectWSS(creds, form)
	defer s.Close()
	defer ws.Close()

	// send notification over http
	TITLE := RandomString(10)
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
	db, _ := DBConn(os.Getenv("db"))
	removeCredKey(db, f.Get("UUID"))
	_, res, _, _ = ConnectWSS(creds, f)
	if res.StatusCode != VALIDCODES["RESET_KEY"] {
		t.Errorf("expected %v got %v", VALIDCODES["RESET_KEY"], res.StatusCode)
	}
}

// if there is no credential_key in the db the client should be able to request new key for same credentials
func TestNewCredentialKey(t *testing.T) {
	var _, f = GenUser() // generate user

	// remove credential_key
	db, _ := DBConn(os.Getenv("db"))
	removeCredKey(db, f.Get("UUID"))

	r := PostRequest("", f, http.HandlerFunc(s.CredentialHandler))
	var newcreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &newcreds)
	if len(newcreds.Key) == 0 || len(newcreds.Value) != 0 {
		t.Error("Error fetching new credentials for user")
	}
}
