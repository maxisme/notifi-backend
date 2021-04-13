package main

import (
	"github.com/maxisme/notifi-backend/crypt"
	url2 "net/url"
	"regexp"
	"strings"

	uuid "github.com/satori/go.uuid"
)

// ValidVersionRegex is regex to match a valid version
var ValidVersionRegex = regexp.MustCompile(`v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?`)

// IsValidVersion checks if a string is in the format of a valid version
func IsValidVersion(version string) bool {
	version = strings.TrimSpace(version)
	if len(version) == 0 {
		return false
	}
	return ValidVersionRegex.MatchString(version)
}

// IsValidUUID checks a string is a UUID
func IsValidUUID(str string) bool {
	_, err := uuid.FromString(str)
	return err == nil
}

// IsValidCredentials checks a string is the length of expected Credentials
func IsValidCredentials(credentials string) bool {
	return len(credentials) == credentialLen
}

// IsValidB64PublicKey checks a string is a valid publicKey
func IsValidB64PublicKey(publicKey string) bool {
	_, err := crypt.B64StringToPubKey(publicKey)
	return err == nil
}

// IsValidURL checks a string is a URL
func IsValidURL(url string) bool {
	if url == "" {
		return true
	}
	_, err := url2.ParseRequestURI(url)
	return err == nil
}
