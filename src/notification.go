package main

import (
	"database/sql"
	"errors"
	"fmt"
	. "github.com/maxisme/notifi-backend/structs"
	tdb "github.com/maxisme/notifi-backend/tracer/db"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/maxisme/notifi-backend/crypt"
)

// restrictions of notifications
const (
	maxTitle       = 1000
	maxMessage     = 10000
	maxImageBytes  = 100000
	imageTimeoutMS = 300
)

// Store will store n Notification in the database
func Store(r *http.Request, db *sql.DB, n Notification) (err error) {
	// language=PostgreSQL
	_, err = tdb.Exec(r, db, `
	INSERT INTO notifications (UUID, title, message, image, link, credentials, time, encrypted_key) 
    VALUES($1, $2, $3, $4, $5, $6, $7, $8)`, n.UUID, n.Title, n.Message, n.Image, n.Link, crypt.Hash(n.Credentials), n.Time, n.EncryptedKey)
	return
}

// Validate runs validation on n Notification
func Validate(r *http.Request, n Notification) error {
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
		client := http.Client{
			Timeout: imageTimeoutMS * time.Millisecond,
		}
		resp, err := client.Head(n.Image)
		if err != nil {
			Log(r, log.InfoLevel, err)
			n.Image = "" // remove image reference
		} else {
			contentLen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
			if err != nil {
				Log(r, log.InfoLevel, err)
				n.Image = "" // remove image reference
			}

			if contentLen > maxImageBytes {
				return fmt.Errorf("Image too large (%d) should be less than %d bytes", contentLen, maxImageBytes)
			}
		}
	}

	return nil
}

// FetchStoredNotifications Fetches all notifications belonging to user.
// Will only decrypt if the user has no public serverKey and thus the messages were encrypted on the Server with AES.
func (u User) FetchStoredNotifications(r *http.Request, db *sql.DB) ([]Notification, error) {
	// language=PostgreSQL
	query := `
	SELECT
		uuid,
		time,
		title, 
		message,
		image,
		link,
	    encrypted_key
	FROM notifications
	WHERE credentials = $1`
	rows, err := tdb.Query(r, db, query, crypt.Hash(u.Credentials.Value))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		var encryptedKey sql.NullString
		err := rows.Scan(&n.UUID, &n.Time, &n.Title, &n.Message, &n.Image, &n.Link, &encryptedKey)
		if err != nil {
			return nil, err
		}
		n.EncryptedKey = encryptedKey.String
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

// IncreaseNotificationCnt increases the notification count in the database of the Credentials from the
// Notification and returns it
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

// FetchUser gets notification count and public key of user
func FetchUser(r *http.Request, db *sql.DB, credentials string) (User, error) {
	// language=PostgreSQL
	query := `
	SELECT notification_cnt, public_key
	FROM users
	WHERE credentials = $1`
	row := tdb.QueryRow(r, db, query, crypt.Hash(credentials))
	var u User
	err := row.Scan(&u.NotificationCnt, &u.PublicKey)
	return u, err
}
