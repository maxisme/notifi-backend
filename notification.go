package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Notification struct {
	ID          int    `json:"id"`
	Credentials string `json:"-"`
	Time        string `json:"time"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	Image       string `json:"image"`
	Link        string `json:"link"`
}

var maxtitle = 1000
var maxmessage = 10000
var maximage = 100000
var key = []byte(os.Getenv("encryption_key"))

func (n Notification) Store(db *sql.DB) error {
	var DBUser User
	err := DBUser.Get(db, n.Credentials)
	if err != nil {
		return err
	}

	n.Title, err = EncryptAES(n.Title, key)
	if err != nil {
		return err
	}

	n.Message, err = EncryptAES(n.Message, key)
	if err != nil {
		return err
	}

	n.Image, err = EncryptAES(n.Image, key)
	if err != nil {
		return err
	}

	n.Link, err = EncryptAES(n.Link, key)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
	INSERT INTO notifications 
    (id, title, message, image, link, credentials) 
    VALUES(?, ?, ?, ?, ?, ?)`, n.ID, n.Title, n.Message, n.Image, n.Link, Hash(n.Credentials))
	return err
}

func (n Notification) Validate() error {
	if len(n.Credentials) == 0 {
		return errors.New("Invalid credentials!")
	}

	if n.Credentials == "<credentials>" {
		return errors.New("You have not set your personal credentials given to you by the notifi app! You instead used the placeholder '<credentials>'!")
	}

	if len(n.Title) == 0 {
		return errors.New("You must enter a title!")
	} else if len(n.Title) > maxtitle {
		return errors.New("You must enter a shorter title!")
	}

	if len(n.Message) > maxmessage {
		return errors.New("You must enter a shorter message!")
	}

	if IsValidURL(n.Link) != nil {
		return errors.New("Invalid URL for link!")
	}

	if IsValidURL(n.Image) != nil {
		return errors.New("Invalid URL for image!")
	}

	if len(n.Image) > 0 {
		if strings.Contains(n.Image, "http://") {
			return errors.New("Image host must use https!")
		}

		timeout := time.Duration(300 * time.Millisecond)
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Head(n.Image)
		if err != nil {
			Handle(err)
			n.Image = "" // remove image reference
		} else {
			contentlen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
			if err != nil {
				Handle(err)
				n.Image = "" // remove image reference
			}

			if contentlen > maximage {
				return errors.New("Image too large (" + string(contentlen) + ") should be less than " + string(maximage))
			}
		}
	}

	return nil
}

// Public Key Encrypt
func (n *Notification) Encrypt() {

}

// AES decrypt notification - only works when user has no public key and the encryption is done
// on the Server TODO only use public key encryption
func (n *Notification) Decrypt() error {
	title, err := DecryptAES(n.Title, key)
	if err != nil {
		return err
	} else {
		n.Title = title
	}

	message, err := DecryptAES(n.Message, key)
	Handle(err)
	if err == nil {
		n.Message = message
	}

	image, err := DecryptAES(n.Image, key)
	Handle(err)
	if err == nil {
		n.Image = image
	}

	link, err := DecryptAES(n.Link, key)
	Handle(err)
	if err == nil {
		n.Link = link
	}
	return err
}

// Fetch all notifications belonging to user. Will only decrypt if the user has no public key and thus
// the messages were encrypted on the Server with AES.
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
	rows, err := db.Query(query, Hash(u.Credentials.Value))
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		err := rows.Scan(&n.ID, &n.Time, &n.Title, &n.Message, &n.Image, &n.Link)
		if err != nil {
			return nil, err
		}

		// if there is no public key decrypt using AES notification
		err = n.Decrypt()

		if err == nil {
			notifications = append(notifications, n)
		} else {
			return nil, err
		}
	}
	return notifications, nil
}

func (u User) DeleteReceivedNotifications(db *sql.DB, ids string) {
	IDArr := []interface{}{Hash(u.Credentials.Value)}

	// validate all comma separated values are integers
	for _, element := range strings.Split(ids, ",") {
		if val, err := strconv.Atoi(element); err != nil {
			log.Println(element + " is not a number!")
			return
		} else {
			IDArr = append(IDArr, val)
		}
	}

	query := `
	DELETE FROM notifications
	WHERE credentials = ?
	AND id IN (?` + strings.Repeat(",?", len(IDArr)-2) + `)`

	_, err := db.Exec(query, IDArr...)
	if err != nil {
		log.Println(err.Error())
	}
}

func IncreaseNotificationCnt(db *sql.DB, credentials string) {
	_, _ = db.Exec(`UPDATE users 
	SET notification_cnt = notification_cnt + 1 WHERE credentials = ?`, Hash(credentials))
	// TODO handle err when not to do with not being able to increase because there are no matching credentials
}

func FetchNumNotifications(db *sql.DB) int {
	id := 0
	rows, _ := db.Query("SELECT SUM(notification_cnt) from users")
	if rows.Next() {
		_ = rows.Scan(&id)
	}
	return id
}
