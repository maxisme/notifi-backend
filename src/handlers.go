package main

import (
	"encoding/json"
	"fmt"
	"github.com/appleboy/go-fcm"
	"github.com/gorilla/websocket"
	"github.com/maxisme/notifi-backend/ws"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
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
		WriteError(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	user := User{
		Credentials: Credentials{
			Value: r.Header.Get("Credentials"),
			Key:   r.Header.Get("Key"),
		},
		UUID:          r.Header.Get("Uuid"),
		AppVersion:    r.Header.Get("Version"),
		FirebaseToken: r.Header.Get("Firebase-Token"),
	}

	// validate inputs
	if !IsValidUUID(user.UUID) {
		WriteHTTPError(w, r, http.StatusBadRequest, "Invalid UUID")
		return
	} else if !IsValidVersion(user.AppVersion) {
		WriteHTTPError(w, r, http.StatusBadRequest, fmt.Sprintf("Invalid Version %v", user.AppVersion))
		return
	} else if !IsValidCredentials(user.Credentials.Value) {
		WriteHTTPError(w, r, http.StatusForbidden, "Invalid Credentials")
		return
	}

	var errorCode = 0
	var DBUser User
	_ = DBUser.GetWithUUID(r, s.db, user.UUID)
	if len(DBUser.Credentials.Key) == 0 {
		errorCode = RequestNewUserCode
		if len(DBUser.Credentials.Value) == 0 {
			Log(r, log.InfoLevel, "No credentials or key for: "+user.UUID)
		} else {
			Log(r, log.InfoLevel, "No credential key for: "+user.UUID)
		}
	} else if !user.Verify(r, s.db) {
		errorCode = http.StatusForbidden
	}

	if errorCode != 0 {
		WriteError(w, "Method not allowed", errorCode)
		return
	}

	if err := user.StoreLogin(r, s.db); err != nil {
		WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// connect to socket
	WSConn, err := ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// initialise WS funnel
	hashedCredentials := crypt.Hash(user.Credentials.Value)
	funnel := &ws.Funnel{
		Key:    hashedCredentials,
		WSConn: WSConn,
		PubSub: s.redis.Subscribe(hashedCredentials),
	}
	s.funnels.Add(s.redis, funnel)

	Log(r, log.InfoLevel, "Client Connected: "+hashedCredentials)

	// send "." to client when successfully connected to web socket
	if err := WSConn.WriteMessage(websocket.TextMessage, []byte(".")); err != nil {
		WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// send all stored notifications from db
	notifications, err := user.FetchNotifications(s.db)
	if err != nil {
		WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	fmt.Println(notifications)
	if len(notifications) > 0 {
		bytes, err := json.Marshal(notifications)
		if err == nil {
			if err := WSConn.WriteMessage(websocket.TextMessage, bytes); err != nil {
				WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
		} else {
			Log(r, log.WarnLevel, err.Error())
		}
	}

	// incoming socket messages
	for {
		_, message, err := WSConn.ReadMessage()
		if err != nil {
			break
		}
		var uuids []string
		if err := json.Unmarshal(message, &uuids); err != nil {
			Log(r, log.WarnLevel, err)
			break
		}
		go func() {
			if err := user.DeleteNotificationsWithIDs(r, s.db, uuids, hashedCredentials); err != nil {
				Log(r, log.WarnLevel, err)
			}
		}()
	}

	if err := s.funnels.Remove(funnel); err != nil {
		Log(r, log.WarnLevel, err)
	}

	Log(r, log.InfoLevel, "Client Disconnected: "+hashedCredentials)

	// close connection
	if err := user.CloseLogin(r, s.db); err != nil {
		Log(r, log.WarnLevel, err)
	}
}

// CredentialHandler is the handler for creating and updating Credentials
func (s *Server) CredentialHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		WriteError(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// create PostUser struct
	PostUser := User{
		UUID:          r.Form.Get("UUID"),
		FirebaseToken: r.Form.Get("firebase_token"),

		// if asking for new Credentials
		Credentials: Credentials{
			Value: r.Form.Get("current_credentials"),
			Key:   r.Form.Get("current_credential_key"),
		},
	}

	if !IsValidUUID(PostUser.UUID) {
		WriteError(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	creds, err := PostUser.Store(r, s.db)
	if err != nil {
		if err.Error() != "pq: duplicate key value violates unique constraint \"uuid\"" {
			Log(r, log.FatalLevel, err.Error()) // TODO test
		}
		WriteHTTPError(w, r, 401, err.Error()) // UUID already exists
		return
	}

	c, err := json.Marshal(creds)
	if err != nil {
		WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	_, err = w.Write(c)
	if err != nil {
		WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
}

// APIHandler is the http handler for handling API calls to create notifications
func (s *Server) APIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "GET" {
		WriteError(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var notification Notification
	if err := decoder.Decode(&notification, r.Form); err != nil {
		WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := notification.Validate(r); err != nil {
		WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// increase notification count
	err := IncreaseNotificationCnt(r, s.db, notification)
	if err != nil {
		// no such user with Credentials
		return
	}

	// set time
	loc, _ := time.LoadLocation("UTC")
	notification.Time = time.Now().In(loc).Format(NotificationTimeLayout)

	notification.UUID = uuid.New().String()
	notificationMsgBytes, err := json.Marshal([]Notification{notification})
	if err != nil {
		WriteHTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	u := User{}
	err = u.Get(r, s.db, notification.Credentials)
	if err != nil {
		WriteHTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if len(u.FirebaseToken) > 0 {
		msg := &fcm.Message{
			To: u.FirebaseToken,
			Notification: &fcm.Notification{
				Title: notification.Title,
				Body:  notification.Message,
			},
		}
		_, err := s.firebaseClient.Send(msg)
		if err != nil {
			Log(r, log.WarnLevel, err)
		}
	}

	err = s.funnels.SendBytes(s.redis, crypt.Hash(notification.Credentials), notificationMsgBytes)
	if err != nil {
		// store as user is not online
		var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
		if err := notification.Store(r, s.db, encryptionKey); err != nil {
			WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}
}

// VersionHandler returns xml information on the latest app version or if passing version GET argument will return
// 200 if there is new version else 404
func (s *Server) VersionHandler(w http.ResponseWriter, r *http.Request) {
	_, develop := r.URL.Query()["develop"]

	githubResponses, err := GetGitHubResponses("https://api.github.com/repos/maxisme/notifi/releases")
	if err != nil {
		WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	var githubResponse GitHubResponse
	for _, githubResp := range githubResponses {
		if !githubResp.Draft {
			if develop && githubResp.Prerelease {
				githubResponse = githubResp
				break
			} else if !develop && !githubResp.Prerelease {
				githubResponse = githubResp
				break
			}
		}
	}

	if currentVersion, found := r.URL.Query()["version"]; found {
		currentV, err := version.NewVersion(currentVersion[0])
		if err != nil {
			panic(err)
		}
		latestV, err := version.NewVersion(githubResponse.TagName)
		if err != nil {
			panic(err)
		}

		if currentV.LessThan(latestV) {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	pubDttm := githubResponse.PublishedAt.Format(rfc2822)
	dmgUrl := githubResponse.Assets[0].BrowserDownloadURL
	w.Header().Set("Content-Type", "application/xml")
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0" xmlns:sparkle="https://notifi.it/xml-namespaces/sparkle" xmlns:dc="https://notifi.it/dc/elements/1.1/">
  <channel>
	<item>
		<title>%s</title>
		<description><![CDATA[
			%s
		]]>
		</description>
		<pubDate>%s</pubDate>
		<enclosure url="%s" sparkle:version="%s"/>
	</item>
  </channel>
</rss>`, githubResponse.Name, githubResponse.Body, pubDttm, dmgUrl, githubResponse.TagName)
	if _, err := w.Write([]byte(xml)); err != nil {
		panic(err)
	}
}
