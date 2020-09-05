package main

import (
	"database/sql"
	"errors"
	"fmt"
	tdb "github.com/maxisme/notifi-backend/tracer/db"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/maxisme/notifi-backend/crypt"
)

// Notification structure
type Notification struct {
	Credentials credentials
	ID          int    `json:"id"`
	Time        string `json:"time"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	Image       string `json:"image"`
	Link        string `json:"link"`
}

// size restrictions of notifications
const (
	maxTitle      = 1000
	maxMessage    = 10000
	maxImageBytes = 100000
)

var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))

// Store will store n Notification in the database after encrypting the content
func (n Notification) Store(r *http.Request, db *sql.DB) (err error) {
	n.Title, err = crypt.EncryptAES(n.Title, encryptionKey)
	if err != nil {
		return
	}

	n.Message, err = crypt.EncryptAES(n.Message, encryptionKey)
	if err != nil {
		return
	}

	n.Image, err = crypt.EncryptAES(n.Image, encryptionKey)
	if err != nil {
		return
	}

	n.Link, err = crypt.EncryptAES(n.Link, encryptionKey)
	if err != nil {
		return
	}

	_, err = tdb.Exec(r, db, `
	INSERT INTO notifications 
    (id, title, message, image, link, credentials) 
    VALUES(?, ?, ?, ?, ?, ?)`, n.ID, n.Title, n.Message, n.Image, n.Link, crypt.Hash(n.Credentials))
	return
}

// Validate runs validation on n Notification
func (n Notification) Validate(r *http.Request) error {
	if len(n.Credentials) == 0 {
		return errors.New("You must specify Credentials!")
	}

	if n.Credentials == "<credentials>" {
		return errors.New("You have not set your personal Credentials given to you by the notifi app! You instead used the placeholder '<Credentials>'!")
	}

	if len(n.Title) == 0 {
		return errors.New("You must enter a title!")
	} else if len(n.Title) > maxTitle {
		return errors.New("You must enter a shorter title!")
	}

	if len(n.Message) > maxMessage {
		return errors.New("You must enter a shorter message!")
	}

	if !IsValidURL(n.Link) {
		return errors.New("Invalid URL for link!")
	}

	if !IsValidURL(n.Image) {
		return errors.New("Invalid URL for image!")
	}

	if len(n.Image) > 0 {
		if strings.Contains(n.Image, "http://") {
			return errors.New("Image host must use https!")
		}

		timeout := 300 * time.Millisecond
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Head(n.Image)
		if err != nil {
			Log(r, log.WarnLevel, err)
			n.Image = "" // remove image reference
		} else {
			contentLen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
			if err != nil {
				Log(r, log.WarnLevel, err)
				n.Image = "" // remove image reference
			}

			if contentLen > maxImageBytes {
				return errors.New(fmt.Sprintf("Image too large (%d) should be less than %d", contentLen, maxImageBytes))
			}
		}
	}

	return nil
}

// Decrypt decrypts n Notification
func (n *Notification) Decrypt() error {
	title, err := crypt.DecryptAES(n.Title, encryptionKey)
	if err == nil {
		n.Title = title
	}

	message, err := crypt.DecryptAES(n.Message, encryptionKey)
	if err == nil {
		n.Message = message
	}

	image, err := crypt.DecryptAES(n.Image, encryptionKey)
	if err == nil {
		n.Image = image
	}

	link, err := crypt.DecryptAES(n.Link, encryptionKey)
	if err == nil {
		n.Link = link
	}
	return err
}

// FetchNotifications Fetches all notifications belonging to user.
// Will only decrypt if the user has no public serverKey and thus the messages were encrypted on the Server with AES.
func (u User) FetchNotifications(db *sql.DB) ([]Notification, error) {
	query := `
	SELECT
		id,
		DATE_FORMAT(time, '%Y-%m-%d %T') as time,
		title, 
		message,
		image,
		link
	FROM notifications
	WHERE credentials = ?`
	rows, err := db.Query(query, crypt.Hash(u.Credentials.Value))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		err := rows.Scan(&n.ID, &n.Time, &n.Title, &n.Message, &n.Image, &n.Link)
		if err != nil {
			return nil, err
		}

		// if there is no public serverKey decrypt using AES notification
		err = n.Decrypt()
		if err == nil {
			notifications = append(notifications, n)
		} else {
			return nil, err
		}
	}
	return notifications, nil
}

// DeleteNotificationsWithIDs deletes all comma separated ids
func (u User) DeleteNotificationsWithIDs(r *http.Request, db *sql.DB, ids string) error {
	// arguments to be passed to the SQL query
	SQLArgs := []interface{}{crypt.Hash(u.Credentials.Value)}

	// validate all comma separated values are integers
	numIds := int64(0)
	for _, element := range strings.Split(ids, ",") {
		if len(element) == 0 {
			continue
		}
		val, err := strconv.Atoi(element)
		if err != nil {
			continue
		}
		SQLArgs = append(SQLArgs, val)
		numIds += 1
	}

	if len(SQLArgs)-2 > 0 {
		query := fmt.Sprintf(`
		DELETE FROM notifications
		WHERE credentials = ?
		AND id IN (?%s)`, strings.Repeat(",?", len(SQLArgs)-2))

		_, err := tdb.Exec(r, db, query, SQLArgs...)
		if err != nil {
			return err
		}
	}
	return nil
}

// IncreaseNotificationCnt increases the notification count in the database of the specific Credentials from the
// Notification
func IncreaseNotificationCnt(r *http.Request, db *sql.DB, n Notification) error {
	// language=MySQL
	res, err := tdb.Exec(r, db, `UPDATE users 
	SET notification_cnt = notification_cnt + 1 WHERE credentials = ?`, crypt.Hash(n.Credentials))
	if err != nil {
		return err
	}
	num, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if num == 0 {
		return errors.New("no such user with credentials")
	}
	return nil
}
