package main

import (
	"errors"
	"fmt"
	"github.com/guregu/dynamo"
	"os"
	"time"
)

var UserTable = os.Getenv("USER_TABLE_NAME")

// User structure
type User struct {
	AppVersion      string    `dynamo:"app_version"`
	Created         time.Time `dynamo:"created_dttm"`
	Credentials     string    `dynamo:"credentials,hash"`
	CredentialsKey  string    `dynamo:"credential_key"`
	ConnectionID    string    `dynamo:"connection_id,hash"`
	OS              string    `dynamo:"operating_system"`
	FirebaseToken   string    `dynamo:"firebase_token,allowempty"`
	LastLogin       time.Time `dynamo:"last_login_dttm"`
	NotificationCnt int       `dynamo:"notification_cnt"`
	UUID            string    `dynamo:"device_uuid,hash"`
}

// Credentials structure
type Credentials struct {
	Value string `json:"credentials"`
	Key   string `json:"credential_key"`
}

const (
	credentialLen    = 25
	credentialKeyLen = 100
)

// Store stores or updates u User with new Credentials depending on whether the user passes current Credentials
// in the u User struct.
func (user User) Store(db *dynamo.DB) (Credentials, error) {
	newCredentials := Credentials{
		RandomString(credentialLen),
		RandomString(credentialKeyLen),
	}

	var StoredUser User
	_ = db.Table(UserTable).Get("device_uuid", Hash(user.UUID)).One(&StoredUser)
	if len(StoredUser.UUID) > 0 {
		if len(StoredUser.CredentialsKey) == 0 && len(StoredUser.Credentials) > 0 {
			StoredUser.CredentialsKey = PassHash(newCredentials.Key)
			if err := db.Table(UserTable).Put(StoredUser).Run(); err != nil {
				return Credentials{}, err
			}
			newCredentials.Value = ""
			return newCredentials, nil
		} else if len(StoredUser.CredentialsKey) == 0 && len(StoredUser.Credentials) == 0 {
			StoredUser.CredentialsKey = PassHash(newCredentials.Key)
			StoredUser.Credentials = Hash(newCredentials.Value)
			if err := db.Table(UserTable).Put(StoredUser).Run(); err != nil {
				return Credentials{}, err
			}
			return newCredentials, nil
		}
	}

	isNewUser := true
	if len(StoredUser.Credentials) > 0 {
		// UUID already exists
		if len(user.CredentialsKey) > 0 && IsValidCredentials(user.Credentials) {
			// If client passes current details they are asking for new Credentials.
			// Verify the Credentials passed are valid
			if user.Verify(StoredUser) {
				isNewUser = false
			} else {
				return Credentials{}, errors.New("unable to create new credentials")
			}
		}
	}

	if isNewUser && len(StoredUser.UUID) > 0 {
		return Credentials{}, fmt.Errorf("UUID (%s) already exists", Hash(user.UUID))
	}

	StoredUser.Credentials = Hash(newCredentials.Value)
	StoredUser.CredentialsKey = PassHash(newCredentials.Key)
	StoredUser.UUID = Hash(user.UUID)
	StoredUser.Created = time.Now()

	// create or update new user
	if err := db.Table(UserTable).Put(StoredUser).Run(); err != nil {
		return Credentials{}, err
	}
	return newCredentials, nil
}

// Verify verifies a u User s credentials
func (user User) Verify(dbUser User) bool {
	isValidKey := VerifyPassHash(dbUser.CredentialsKey, user.CredentialsKey)
	isValidUUID := dbUser.UUID == Hash(user.UUID)
	return isValidKey && isValidUUID
}
