package main

import (
	"github.com/maxisme/notifi-backend/crypt"
	"testing"
)

func TestInvalidRequiredEnvs(t *testing.T) {
	envKey := crypt.RandomString(100)
	err := RequiredEnvs([]string{envKey})
	if err == nil {
		t.Errorf("Should have been no environemnt variable for serverkey '%s'", envKey)
	}
}
