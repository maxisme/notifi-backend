package main

import (
	uuid "github.com/satori/go.uuid"
	url2 "net/url"
	"regexp"
	"strings"
)

var ValidVersionRegex *regexp.Regexp

func init() {
	ValidVersionRegex = regexp.MustCompile(`^[\d\.]*$`)
}

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

// IsValidCredentials checks a string is the length of expected credentials
func IsValidCredentials(credentials string) bool {
	return len(credentials) == CredentialLen
}

// IsValidURL checks a string is a URL
func IsValidURL(url string) bool {
	if url == "" {
		return true
	}
	_, err := url2.ParseRequestURI(url)
	return err == nil
}
