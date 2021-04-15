package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/maxisme/notifi-backend/conn"
	"github.com/maxisme/notifi-backend/structs"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxisme/notifi-backend/ws"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxisme/notifi-backend/crypt"
)

var s Server

// applied to every test
func TestMain(t *testing.M) {

	// init server db connection
	db, err := conn.PgConn()
	if err != nil {
		panic(err)
	}

	// init server redis connection
	red, err := conn.RedisConn()
	if err != nil {
		panic(err)
	}

	s = Server{
		db:        db,
		redis:     red,
		funnels:   &ws.Funnels{Clients: make(map[string]*ws.Funnel)},
		serverKey: "rps2P8irs0mT5uCgicv8m5PMq9a6WyzbxL7HWeRK",
	}

	code := t.Run() // RUN THE TEST

	// after individual test
	os.Exit(code)
}

/////////////
// helpers //
/////////////
func PostRequest(url string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Sec-Key", os.Getenv("SERVER_KEY"))

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

func connectWSS(creds Credentials, form url.Values) (*httptest.Server, *http.Response, *websocket.Conn, error) {
	wsheader := http.Header{}
	wsheader.Add("Credentials", creds.Value)
	wsheader.Add("Key", creds.Key)
	wsheader.Add("Uuid", form.Get("UUID"))
	wsheader.Add("Version", "1.0")

	return connectWSSHeader(wsheader)
}

func connectWSSHeader(wsheader http.Header) (*httptest.Server, *http.Response, *websocket.Conn, error) {
	server := httptest.NewServer(http.HandlerFunc(s.WSHandler))
	ws, res, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), wsheader)
	if err == nil {
		// add ws read timeout
		_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	}
	return server, res, ws, err
}

func SendNotification(credentials string, title string) *httptest.ResponseRecorder {
	form := url.Values{}
	form.Add("Credentials", credentials)
	form.Add("title", title)
	req, _ := http.NewRequest("GET", "/api?"+form.Encode(), nil)
	rr := httptest.NewRecorder()
	http.HandlerFunc(s.APIHandler).ServeHTTP(rr, req)
	return rr
}

func removeUserCredKey(db *sql.DB, UUID string) {
	_, err := db.Exec(`UPDATE users
	SET credential_key = NULL
	WHERE UUID=$1`, crypt.Hash(UUID))
	if err != nil {
		panic(err)
	}
}

func removeUserCreds(db *sql.DB, UUID string) {
	_, err := db.Exec(`UPDATE users
	SET credential_key = NULL, credentials = NULL
	WHERE UUID=$1`, crypt.Hash(UUID))
	if err != nil {
		panic(err)
	}
}

func uuidInNotificationsExists(db *sql.DB, UUID string) bool {
	dbUUID := ""
	res := db.QueryRow(`SELECT uuid
	FROM notifications
	WHERE uuid=$1`, UUID)
	_ = res.Scan(&dbUUID)
	fmt.Println("db:" + dbUUID)
	return dbUUID == UUID
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
	if r.Code != 200 {
		t.Errorf("handler returned wrong status code: got %d want %d", r.Code, 200)
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
		{"Sec-Key", os.Getenv("SERVER_KEY"), false},
		{"Credentials", creds.Value, false},
		{"Key", creds.Key, false},
		{"Uuid", form.Get("UUID"), false},
		{"Version", "1.0.1", true},
	}

	for _, tt := range headers {
		wsheader.Add(tt.key, tt.value)
		server, _, WS, err := connectWSSHeader(wsheader)
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
	_, _, WS, _ := connectWSS(creds, uform)
	defer WS.Close()

	// verify message was sent now connected
	notifications := readNotifications(WS)
	if notifications[0].Title != TITLE {
		t.Error("Incorrect title returned!")
	}
}

func readNotifications(ws *websocket.Conn) (notifications []structs.Notification) {
	_, mess, err := ws.ReadMessage()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = json.Unmarshal(mess, &notifications)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	return
}

func TestReceivingNotificationWSOnline(t *testing.T) {
	var creds, form = GenUser() // generate user

	// connect to ws
	_, _, WS, _ := connectWSS(creds, form)
	defer WS.Close()

	// send notification over http
	TITLE := crypt.RandomString(10)
	SendNotification(creds.Value, TITLE)

	// read notification over ws
	notifications := readNotifications(WS)
	if notifications == nil {
		t.Errorf("No notifications!")
		return
	}
	if notifications[0].Title != TITLE {
		t.Errorf("Titles do not match! %v - %v", notifications[0].Title, TITLE)
	}
}

func TestDeleteNotificationsOnReceiveWS(t *testing.T) {
	var creds, form = GenUser() // generate user

	// send notification over http
	TITLE := crypt.RandomString(10)
	SendNotification(creds.Value, TITLE)

	// connect to ws
	_, _, WS, _ := connectWSS(creds, form)
	defer WS.Close()

	// read notification over ws
	notifications := readNotifications(WS)
	if notifications == nil {
		t.Errorf("No notifications!")
		return
	}

	uuid := notifications[0].UUID
	uuids := []string{uuid}
	uuidsJson, _ := json.Marshal(uuids)

	// verify notification is in DB
	if !uuidInNotificationsExists(s.db, uuid) {
		t.Errorf("Missing uuid!")
	}

	print(uuids)
	_ = WS.WriteMessage(websocket.TextMessage, uuidsJson)

	time.Sleep(10 * time.Millisecond)
	if uuidInNotificationsExists(s.db, uuid) {
		t.Errorf("uuid should have been deleted")
	}

	// verify notification is not in DB
}

func TestWSSResetKey(t *testing.T) {
	var creds, f = GenUser() // generate user
	_, res, _, _ := connectWSS(creds, f)
	if res.StatusCode != 101 {
		t.Errorf("expected %v got %v", 101, res.StatusCode)
	}

	// remove credential_key
	removeUserCredKey(s.db, f.Get("UUID"))
	_, res, _, _ = connectWSS(creds, f)
	if res.StatusCode != RequestNewUserCode {
		t.Errorf("expected %v got %v", RequestNewUserCode, res.StatusCode)
	}
}

// if there is no UUID in the db the client should be able to request new serverKey
func TestWSSNoUUID(t *testing.T) {
	var creds, f = GenUser() // generate user
	_, res, _, _ := connectWSS(creds, f)
	if res.StatusCode != 101 {
		t.Errorf("expected %v got %v", 101, res.StatusCode)
	}

	// remove credential_key
	removeUserCreds(s.db, f.Get("UUID"))
	_, res, _, _ = connectWSS(creds, f)
	if res.StatusCode != RequestNewUserCode {
		t.Errorf("expected %v got %v", RequestNewUserCode, res.StatusCode)
	}
}

// if there is no credential_key in the db the client should be able to request new serverKey for same Credentials
// and receive a new credential serverKey only
func TestRemovedCredentialKey(t *testing.T) {
	var _, f = GenUser() // generate user

	// remove credential_key
	removeUserCredKey(s.db, f.Get("UUID"))

	r := PostRequest("", f, s.CredentialHandler)
	var newCreds Credentials
	_ = json.Unmarshal(r.Body.Bytes(), &newCreds)
	if len(newCreds.Key) == 0 || len(newCreds.Value) != 0 {
		t.Errorf("Error fetching new Credentials for user %v. Expected new serverKey", newCreds)
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

	// expects a new credential serverKey to be returned only
	if len(newCreds.Key) == 0 || len(newCreds.Value) == 0 {
		t.Errorf("Error fetching new Credentials for user %v. Expected new Credentials and serverKey", newCreds)
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
		t.Run(fmt.Sprint(i), func(t *testing.T) {
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
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			req, _ := http.NewRequest(tt.wrongMethod, "", nil)
			rr := httptest.NewRecorder()
			tt.handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("Should have responded with error code %d not %d", http.StatusBadRequest, rr.Code)
			}
		})
	}
}
