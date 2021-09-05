package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/appleboy/go-fcm"
	"github.com/gorilla/websocket"
	. "github.com/maxisme/notifi-backend/logging"
	"github.com/maxisme/notifi-backend/ws"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
)

// custom error codes
const (
	RequestNewUserCode = 551
)

// NotificationTimeLayout layout for the time Format()
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
		UUID:       r.Header.Get("Uuid"),
		AppVersion: r.Header.Get("Version"),
	}

	if len(r.Header.Get("Firebase-Token")) > 0 {
		user.FirebaseToken = sql.NullString{String: r.Header.Get("Firebase-Token"), Valid: true}
	}

	// validate inputs
	if !IsValidUUID(user.UUID) {
		WriteHTTPError(w, r, http.StatusBadRequest, fmt.Sprintf("Invalid UUID: '%s'", user.UUID))
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
	channel := GetWSChannelKey(user.Credentials.Value)
	funnel := &ws.Funnel{
		Channel: channel,
		WSConn:  WSConn,
		PubSub:  s.redis.Subscribe(channel),
	}

	if err := s.funnels.Add(r, s.redis, funnel); err != nil {
		WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// send "." to client when successfully connected to web socket
	if err := WSConn.WriteMessage(websocket.TextMessage, []byte(".")); err != nil {
		WriteHTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// send all stored notifications from db
	if err := user.SendStoredNotifications(r, s.db, WSConn); err != nil {
		Log(r, log.ErrorLevel, err)
	}

	// incoming socket messages
	for {
		_, message, err := WSConn.ReadMessage()
		if err != nil {
			break
		}

		if string(message) == "." {
			if err := user.SendStoredNotifications(r, s.db, WSConn); err != nil {
				Log(r, log.ErrorLevel, err)
			}
			continue
		}
		go func() {
			var uuids []string
			if err := json.Unmarshal(message, &uuids); err != nil {
				Log(r, log.WarnLevel, err)
			}
			if err := user.DeleteNotificationsWithIDs(r, s.db, uuids, user.Credentials.Value); err != nil {
				Log(r, log.InfoLevel, err)
			}
		}()
	}

	if err := s.funnels.Remove(funnel); err != nil {
		Log(r, log.WarnLevel, err)
	}

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
		UUID: r.Form.Get("UUID"),

		// if asking for new Credentials
		Credentials: Credentials{
			Value: r.Form.Get("current_credentials"),
			Key:   r.Form.Get("current_credential_key"),
		},
	}

	if len(r.Form.Get("firebase_token")) > 0 {
		PostUser.FirebaseToken = sql.NullString{String: r.Form.Get("firebase_token"), Valid: true}
	}

	if !IsValidUUID(PostUser.UUID) {
		WriteError(w, fmt.Sprintf("Invalid UUID: '%s'", PostUser.UUID), http.StatusBadRequest)
		return
	}

	creds, err := PostUser.Store(r, s.db)
	if err != nil {
		if err.Error() != "pq: duplicate key value violates unique constraint \"uuid\"" {
			Log(r, log.InfoLevel, err.Error())
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

	if u.FirebaseToken.Valid {
		msg := &fcm.Message{
			To: u.FirebaseToken.String,
			Notification: &fcm.Notification{
				Title: notification.Title,
				Body:  notification.Message,
				Sound: "default",
			},
		}
		resp, err := s.firebaseClient.Send(msg)
		if err != nil {
			Log(r, log.WarnLevel, err)
		}
		if resp.Error != nil {
			Log(r, log.WarnLevel, resp.Error)
		}
	}

	err = s.funnels.SendBytes(r, s.redis, GetWSChannelKey(notification.Credentials), notificationMsgBytes)
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
