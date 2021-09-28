package main

import (
	"testing"
)

func TestIsNotValidUUID(t *testing.T) {
	UUID := "62b5873e-71bf-4659-af9d796581f126f8"
	if IsValidUUID(UUID) {
		t.Errorf("'%s' should not be a valid UUID", UUID)
	}
}

func TestIsValidUUID(t *testing.T) {
	UUID := "BB8C9950-286C-5462-885C-0CFED585423B"
	if !IsValidUUID(UUID) {
		t.Errorf("'%s' should be a valid UUID", UUID)
	}
}

var versionTests = []struct {
	in  string
	out bool
}{
	{"", false},
	{" ", false},
	{"a", false},
	{"1", true},
	{"1.", true},
	{"1.2", true},
	{"1.2aa2.3a", true},
	{"1.a.3", true},
	{"1.2.3", true},
}

func TestVersionValidity(t *testing.T) {
	for _, tt := range versionTests {
		t.Run(tt.in, func(t *testing.T) {
			v := IsValidVersion(tt.in)
			if v != tt.out {
				t.Errorf("got %v, wanted %v", v, tt.out)
			}
		})
	}
}

func TestIsValidCredentials(t *testing.T) {
	credentials := RandomString(credentialLen)
	if !IsValidCredentials(credentials) {
		t.Errorf("'%s' should have been valid Credentials", credentials)
	}
}

func TestIsValidURL(t *testing.T) {
	URL := "https://notifi.it/"
	if !IsValidURL(URL) {
		t.Errorf("'%s' should have been a valid URL", URL)
	}

	URL2 := ""
	if !IsValidURL(URL2) {
		t.Errorf("'%s' should have been a valid URL", URL)
	}
}

var urlTests = []struct {
	in  string
	out bool
}{
	{"", true},
	{" ", false},
	{"foo", false},
	{"notifi.it", false},
	{"https://notifi.it", true},
	{"https://notifi.it/", true},
	{"https://notifi.it/hello/foo", true},
	{"https://notifi.it/hello/foo?12321&nfdjndsalnadfsjn=123", true},
}

func TestURLValidity(t *testing.T) {
	for _, tt := range urlTests {
		t.Run(tt.in, func(t *testing.T) {
			v := IsValidURL(tt.in)
			if v != tt.out {
				t.Errorf("'%s' got %v, wanted %v", tt.in, v, tt.out)
			}
		})
	}
}
