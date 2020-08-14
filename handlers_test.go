package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maxisme/notifi-backend/conn"

	"github.com/maxisme/notifi-backend/ws"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxisme/notifi-backend/crypt"
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

	rr := PostRequest("", form, s.CredentialHandler)
	var creds Credentials
	_ = json.Unmarshal(rr.Body.Bytes(), &creds)
	return creds, form
}

func ConnectWSS(creds Credentials, form url.Values) (*httptest.Server, *http.Response, *websocket.Conn, error) {
	wsheader := http.Header{}
	wsheader.Add("Sec-Key", os.Getenv("server_key"))
	wsheader.Add("Credentials", creds.Value)
	wsheader.Add("Key", creds.Key)
	wsheader.Add("Uuid", form.Get("UUID"))
	wsheader.Add("Version", "1.0")

	return ConnectWSSHeader(wsheader)
}

func ConnectWSSHeader(wsheader http.Header) (*httptest.Server, *http.Response, *websocket.Conn, error) {
	s := httptest.NewServer(http.HandlerFunc(s.WSHandler))
	WS, res, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http"), wsheader)
	if err == nil {
		_ = WS.SetReadDeadline(time.Now().Add(1 * time.Second)) // add timeout
	}
	return s, res, WS, err
}

func SendNotification(credentials string, title string) *httptest.ResponseRecorder {
	nform := url.Values{}
	nform.Add("Credentials", credentials)
	nform.Add("title", title)
	req, _ := http.NewRequest("GET", "/api?"+nform.Encode(), nil)
	rr := httptest.NewRecorder()
	http.HandlerFunc(s.APIHandler).ServeHTTP(rr, req)
	return rr
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
	err := RequiredEnvs([]string{"db", "redis", "encryption_key", "server_key"})
	if err != nil {
		panic(err)
	}

	// create database
	db, err := conn.MysqlConn(os.Getenv("db") + "/?multiStatements=True")
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
	if err := m.Up(); err != nil {
		panic(err)
	}
	if err := m.Down(); err != nil {
		panic(err)
	}
	if err := m.Up(); err != nil {
		panic(err)
	}

	// init server db connection
	db, err = conn.MysqlConn(dbConnStr)
	if err != nil {
		panic(err)
	}

	// init server redis connection
	red, err := conn.RedisConn(os.Getenv("redis"), os.Getenv("redis_db"))
	if err != nil {
		panic(err)
	}

	s = Server{
		db:        db,
		redis:     red,
		funnels:   &ws.Funnels{Clients: make(map[credentials]*ws.Funnel)},
		serverkey: os.Getenv("server_key"),
	}

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
		return
	}

	// try create a new user without specifying current credentials
	r := PostRequest("", form, s.CredentialHandler)
	var nocreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &nocreds)
	if len(nocreds.Value) != 0 || len(nocreds.Key) != 0 {
		t.Errorf("Shouldn't have been able to generate new creds for user! %v", nocreds)
		return
	}

	// ask for new Credentials for user
	form.Add("current_credentials", creds.Value)
	form.Add("current_credential_key", creds.Key)
	r = PostRequest("", form, s.CredentialHandler)
	var newcreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &newcreds)
	if len(newcreds.Value) == 0 || creds.Value == newcreds.Value {
		t.Errorf("Error creating new credentials for user")
	}
}

func TestAddNotification(t *testing.T) {
	var creds, _ = GenUser()

	form := url.Values{}
	form.Add("Credentials", creds.Value)
	form.Add("title", crypt.RandomString(10))

	// POST test
	r := PostRequest("", form, s.APIHandler)
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
	form.Add("Credentials", creds.Value)

	r := PostRequest("", form, s.APIHandler)
	expectedStatus := "You must enter a title!"
	status := strings.TrimSpace(r.Body.String())
	if status != expectedStatus {
		t.Errorf("handler returned wrong status code: got '%v' want '%v'", status, expectedStatus)
	}
}

func TestAddNotificationWithInvalidCredentials(t *testing.T) {
	form := url.Values{}
	form.Add("title", "test")
	form.Add("Credentials", crypt.RandomString(credentialLen))

	r := PostRequest("", form, s.APIHandler)
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
		{"Key", creds.Key, false},
		{"Uuid", form.Get("UUID"), false},
		{"Version", "1.0.1", true},
	}

	for _, tt := range headers {
		wsheader.Add(tt.key, tt.value)
		server, _, WS, err := ConnectWSSHeader(wsheader)
		if err == nil != tt.out {
			println(tt.key + " " + tt.value)
			t.Errorf("got %v, wanted %v", err == nil, tt.out)
		}
		if WS != nil {
			WS.Close()
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
	_, _, WS, _ := ConnectWSS(creds, uform)
	defer WS.Close()

	funnels := ws.Funnels{
		Clients: make(map[string]*ws.Funnel),
		RWMutex: sync.RWMutex{},
	}

	funnel := &ws.Funnel{
		Key:    creds.Value,
		WSConn: WS,
		RDB:    s.redis,
	}
	funnels.Add(funnel)

	// fetch stored notifications on Server that were sent when not connected
	err := funnel.Run(func(messageType int, p []byte, err error) error {
		var notifications []Notification
		_ = json.Unmarshal(p, &notifications)

		if notifications[0].Title != TITLE {
			t.Error("Incorrect title returned!")
		}
		return nil
	}, true)
	if err != nil {
		t.Error(err)
	}
}

func TestReceivingNotificationWSOnline(t *testing.T) {
	var creds, form = GenUser() // generate user

	// connect to ws
	_, _, WS, _ := ConnectWSS(creds, form)
	defer WS.Close()

	funnels := ws.Funnels{
		Clients: make(map[string]*ws.Funnel),
		RWMutex: sync.RWMutex{},
	}

	funnel := &ws.Funnel{
		Key:    creds.Value,
		WSConn: WS,
		RDB:    s.redis,
	}

	funnels.Add(funnel)

	// send notification over http
	TITLE := crypt.RandomString(10)
	SendNotification(creds.Value, TITLE)

	// read notification over ws
	err := funnel.Run(func(messageType int, p []byte, err error) error {
		var notifications []Notification
		err = json.Unmarshal(p, &notifications)
		if err != nil {
			t.Errorf(err.Error())
			return err
		}

		if notifications[0].Title != TITLE {
			t.Errorf("Titles do not match! %v - %v", notifications[0].Title, TITLE)
		}
		return nil
	}, true)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestWSSResetKey(t *testing.T) {
	var creds, f = GenUser() // generate user
	_, res, _, _ := ConnectWSS(creds, f)
	if res.StatusCode != 101 {
		t.Errorf("expected %v got %v", 101, res.StatusCode)
	}

	// remove credential_key
	removeUserCredKey(s.db, f.Get("UUID"))
	_, res, _, _ = ConnectWSS(creds, f)
	if res.StatusCode != RequestNewUserCode {
		t.Errorf("expected %v got %v", RequestNewUserCode, res.StatusCode)
	}
}

// if there is no UUID in the db the client should be able to request new serverkey
func TestWSSNoUUID(t *testing.T) {
	var creds, f = GenUser() // generate user
	_, res, _, _ := ConnectWSS(creds, f)
	if res.StatusCode != 101 {
		t.Errorf("expected %v got %v", 101, res.StatusCode)
	}

	// remove credential_key
	removeUserCreds(s.db, f.Get("UUID"))
	_, res, _, _ = ConnectWSS(creds, f)
	if res.StatusCode != RequestNewUserCode {
		t.Errorf("expected %v got %v", RequestNewUserCode, res.StatusCode)
	}
}

// if there is no credential_key in the db the client should be able to request new serverkey for same Credentials
// and receive a new credential serverkey only
func TestRemovedCredentialKey(t *testing.T) {
	var _, f = GenUser() // generate user

	// remove credential_key
	removeUserCredKey(s.db, f.Get("UUID"))

	r := PostRequest("", f, s.CredentialHandler)
	var newCreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &newCreds)
	if len(newCreds.Key) == 0 || len(newCreds.Value) != 0 {
		t.Errorf("Error fetching new Credentials for user %v. Expected new serverkey", newCreds)
	}
}

// if there is no Credentials and credential_key in the db the UUID user should
func TestRemovedCredentials(t *testing.T) {
	var _, f = GenUser() // generate user

	// remove Credentials and credential_key
	removeUserCreds(s.db, f.Get("UUID"))

	r := PostRequest("", f, s.CredentialHandler)
	var newCreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &newCreds)

	// expects a new credential serverkey to be returned only
	if len(newCreds.Key) == 0 || len(newCreds.Value) == 0 {
		t.Errorf("Error fetching new Credentials for user %v. Expected new Credentials and serverkey", newCreds)
	}
}

var invalidHandlerMethods = []struct {
	handler       http.HandlerFunc
	invalidMethod string
}{
	{s.CredentialHandler, "GET"},
	{s.APIHandler, "PUT"},
	{s.WSHandler, "POST"},
}

// request handlers with incorrect methods
func TestInvalidHandlerMethods(t *testing.T) {
	for i, tt := range invalidHandlerMethods {
		t.Run(string(i), func(t *testing.T) {
			req, _ := http.NewRequest(tt.invalidMethod, "", nil)

			rr := httptest.NewRecorder()
			tt.handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("Should have responded with error code %d not %d", http.StatusBadRequest, rr.Code)
			}
		})
	}
}

var secKeyHandlers = []struct {
	handler     http.HandlerFunc
	wrongMethod string
}{
	{s.CredentialHandler, "GET"},
	{s.WSHandler, "POST"},
}

// test handlers with correct methods but no secret server_key
func TestMissingSecKeyHandlers(t *testing.T) {
	for i, tt := range secKeyHandlers {
		t.Run(string(i), func(t *testing.T) {
			req, _ := http.NewRequest(tt.wrongMethod, "", nil)
			rr := httptest.NewRecorder()
			tt.handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("Should have responded with error code %d not %d", http.StatusBadRequest, rr.Code)
			}
		})
	}
}
