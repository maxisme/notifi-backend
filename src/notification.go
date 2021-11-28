package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/guregu/dynamo"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const MaxNotificationSizeKB = 15

var NotificationTable = os.Getenv("NOTIFICATION_TABLE_NAME")

// Notification structure
type Notification struct {
	Credentials string `json:"-" dynamo:"credentials,hash"`
	Image       string `json:"image" dynamo:"image,allowempty"`
	Link        string `json:"link" dynamo:"link,allowempty"`
	Message     string `json:"message" dynamo:"message,allowempty"`
	Time        string `json:"time" dynamo:"time"`
	Title       string `json:"title" dynamo:"title"`
	UUID        string `json:"UUID" dynamo:"uuid,hash"`
}

// size restrictions of notifications
const (
	maxTitle      = 1000
	maxMessage    = 10000
	maxImageBytes = 2000000 // 2MB
)

const notificationTimeLayout = "2006-01-02 15:04:05"

// Store will store n Notification in the database after encrypting the content
func (n *Notification) Store(db *dynamo.DB, encryptionKey []byte) (err error) {
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

	return db.Table(NotificationTable).Put(&n).Run()
}

// Validate runs validation on n Notification
func (n *Notification) Validate() error {
	if len(n.Credentials) == 0 {
		return errors.New("You must specify Credentials!")
	}

	if n.Credentials == "<credentials>" {
		return errors.New(`You have not set your personal 
		credentials given to you by the notifi app! 
		You instead used the placeholder '<credentials>'`)
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

	sizeKB := n.SizeKB()
	if sizeKB > MaxNotificationSizeKB {
		return fmt.Errorf("Notification too large (%dkb) should be less than %dkb", sizeKB, MaxNotificationSizeKB)
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

// Init set UUID and time
func (n *Notification) Init() {
	loc, _ := time.LoadLocation("UTC")
	n.Time = time.Now().In(loc).Format(notificationTimeLayout)
	n.UUID = uuid.New().String()
}

func (n *Notification) SizeKB() int {
	return binary.Size(reflect.ValueOf(n)) / 1024.0
}
