package main

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/guregu/dynamo"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Notification structure
type Notification struct {
	Credentials string
	UUID        string `json:"UUID"`
	Time        string `json:"time"`
	Title       string `json:"title" `
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

const NotificationTable = "notification"

// Store will store n Notification in the database after encrypting the content
func (n Notification) Store(db *dynamo.DB, encryptionKey []byte) (err error) {
	n.Title, err = EncryptAES(n.Title, encryptionKey)
	if err != nil {
		return
	}

	n.Message, err = EncryptAES(n.Message, encryptionKey)
	if err != nil {
		return
	}

	n.Image, err = EncryptAES(n.Image, encryptionKey)
	if err != nil {
		return
	}

	n.Link, err = EncryptAES(n.Link, encryptionKey)
	if err != nil {
		return
	}

	err = AddItem(db, NotificationTable, n)
	return
}

// Validate runs validation on n Notification
func (n *Notification) Validate() error {
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

		timeout := 500 * time.Millisecond
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Head(n.Image)
		if err != nil {
			n.Image = "" // remove image reference
		} else {
			contentLen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
			if err != nil {
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
	title, err := DecryptAES(n.Title, encryptionKey)
	if err == nil {
		n.Title = title
	}

	message, err := DecryptAES(n.Message, encryptionKey)
	if err == nil {
		n.Message = message
	}

	image, err := DecryptAES(n.Image, encryptionKey)
	if err == nil {
		n.Image = image
	}

	link, err := DecryptAES(n.Link, encryptionKey)
	if err == nil {
		n.Link = link
	}
	return err
}

// IncreaseNotificationCnt increases user notification count
func IncreaseNotificationCnt(db *dynamo.DB, n Notification) error {
	table := db.Table(UserTable)
	wrtx := db.WriteTx()
	rtx := db.GetTx()

	var user User
	getUserQuery := table.Get("credentials", Hash(n.Credentials)).Index("credentials")
	err := rtx.GetOne(getUserQuery, user).Run()
	if err != nil {
		// likely means there is no such user
		return err
	}

	user.NotificationCnt = user.NotificationCnt + 1

	updateUserQuery := table.Update(user.UUID, user)
	return wrtx.Update(updateUserQuery).Run()
}

// Init set UUID and time
func (n *Notification) Init() {
	const notificationTimeLayout = "2006-01-02 15:04:05"

	loc, _ := time.LoadLocation("UTC")
	n.Time = time.Now().In(loc).Format(notificationTimeLayout)
	n.UUID = uuid.New().String()
}
