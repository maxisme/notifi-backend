package main

import (
	uuid "github.com/satori/go.uuid"
	url2 "net/url"
	"regexp"
	"strings"
)

var VALIDVERSIONREGEX = regexp.MustCompile(`^[\d\.]*$`)

func IsValidVersion(version string) bool {
	version = strings.TrimSpace(version)
	if len(version) == 0 {
		return false
	}
	return VALIDVERSIONREGEX.MatchString(version)
}

func IsValidUUID(str string) bool {
	_, err := uuid.FromString(str)
	return err == nil
}

func IsValidCredentials(credentials string) bool {
	return len(credentials) == CREDENTIALLEN
}

func IsValidURL(url string) error {
	if url == "" {
		return nil
	}
	_, err := url2.ParseRequestURI(url)
	return err
}
