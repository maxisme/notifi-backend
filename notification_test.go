package main

import (
	"github.com/maxisme/notifi-backend/crypt"
	"testing"
)

// return error if validation passes
func validateNotificationTest(t *testing.T, n Notification) {
	err := n.Validate()
	if err == nil {
		t.Errorf("Should have returned error")
	}
}

func TestValidation(t *testing.T) {
	n := Notification{}
	validateNotificationTest(t, n)

	n.credentials = "<credentials>" // invalid
	validateNotificationTest(t, n)
	n.credentials = crypt.RandomString(25) // valid

	validateNotificationTest(t, n)

	n.Title = crypt.RandomString(maxTitle + 1) // invalid
	validateNotificationTest(t, n)
	n.Title = "foo" // valid

	n.Message = crypt.RandomString(maxMessage + 1) // invalid
	validateNotificationTest(t, n)
	n.Message = "hi" // valid

	n.Link = "notifi.it" // invalid
	validateNotificationTest(t, n)
	n.Link = "https://notifi.it" // valid

	n.Image = "foo"
	validateNotificationTest(t, n)

	n.Image = "http://notifi.it/images/logo.png" // invalid (not https)
	validateNotificationTest(t, n)

	// TODO handle really large image > maxImageBytes
}
