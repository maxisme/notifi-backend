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
	UUID        string `json:"UUID"`
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
	maxImageBytes = 2000000 // 2MB
)

// Store will store n Notification in the database after encrypting the content
func (n Notification) Store(r *http.Request, db *sql.DB, encryptionKey []byte) (err error) {
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

	// language=PostgreSQL
	_, err = tdb.Exec(r, db, `
	INSERT INTO notifications (UUID, title, message, image, link, credentials, time) 
    VALUES($1, $2, $3, $4, $5, $6, $7)`, n.UUID, n.Title, n.Message, n.Image, n.Link, crypt.Hash(n.Credentials), n.Time)
	return
}

// Validate runs validation on n Notification
func (n Notification) Validate(r *http.Request) error {
	if len(n.Credentials) == 0 {
		return errors.New("You must specify Credentials!")
	}

	if n.Credentials == "<credentials>" {
		return errors.New(`You have not set your personal 
		credentials given to you by the notifi app! 
		You instead used the placeholder '<Credentials>'`)
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
				return fmt.Errorf("Image too large (%d) should be less than %d", contentLen, maxImageBytes)
			}
		}
	}

	return nil
}

// Decrypt decrypts n Notification
func (n *Notification) Decrypt(encryptionKey []byte) error {
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
		uuid,
		time,
		title, 
		message,
		image,
		link
	FROM notifications
	WHERE credentials = $1`
	rows, err := db.Query(query, crypt.Hash(u.Credentials.Value))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		err := rows.Scan(&n.UUID, &n.Time, &n.Title, &n.Message, &n.Image, &n.Link)
		if err != nil {
			return nil, err
		}

		var encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
		err = n.Decrypt(encryptionKey)
		if err != nil {
			Log(nil, log.WarnLevel, err.Error())
			continue
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

// DeleteNotificationsWithIDs deletes comma separated notifications ids
func (u User) DeleteNotificationsWithIDs(r *http.Request, db *sql.DB, ids []string, hashedCredentials string) error {
	for _, UUID := range ids {
		// language=PostgreSQL
		_, err := tdb.Exec(r, db, `DELETE FROM notifications
		WHERE credentials = $1
		AND UUID = $2`, hashedCredentials, UUID)
		if err != nil {
			return err
		}
	}
	return nil
}

// IncreaseNotificationCnt increases user notification count
func IncreaseNotificationCnt(r *http.Request, db *sql.DB, n Notification) error {
	// language=PostgreSQL
	res, err := tdb.Exec(r, db, `UPDATE users 
	SET notification_cnt = notification_cnt + 1 WHERE credentials = $1`, crypt.Hash(n.Credentials))
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
