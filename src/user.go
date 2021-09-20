package main

import (
	"github.com/go-errors/errors"
	"github.com/guregu/dynamo"
	"time"
)

// User structure
type User struct {
	Credentials     Credentials
	ConnectionID    string    `dynamo:"connection_id,allowempty"`
	Created         time.Time `dynamo:"created_dttm"`
	LastLogin       time.Time `dynamo:"last_login_dttm"`
	AppVersion      string    `dynamo:"app_version"`
	NotificationCnt int       `dynamo:"notification_cnt"`
	UUID            string    `dynamo:"device_UUID"`
	Device          string    `dynamo:"device_info"`
	FirebaseToken   string    `dynamo:"firebase_token,allowempty"`
}

// Credentials structure
type credentials = string
type Credentials struct {
	Value credentials `json:"credentials" dynamo:"credentials,hash"`
	Key   string      `json:"credential_key" dynamo:"credential_key"`
}

const (
	credentialLen    = 25
	credentialKeyLen = 100
)
const UserTable = "user"

// Store stores or updates u User with new Credentials depending on whether the user passes current Credentials
// in the u User struct. TODO badly structured separate update and store

func (u User) Store(db *dynamo.DB) (Credentials, error) {
	// create new credentials
	creds := Credentials{
		RandomString(credentialLen),
		RandomString(credentialKeyLen),
	}

	result, _ := GetItem(db, UserTable, "uuid", u.UUID)
	DBUser, ok := result.(User)
	if ok && len(DBUser.UUID) > 0 {
		if len(DBUser.Credentials.Key) == 0 && len(DBUser.Credentials.Value) > 0 {
			DBUser.Credentials.Key = PassHash(creds.Key)
			if err := UpdateItem(db, UserTable, DBUser.Credentials.Value, DBUser); err != nil {
				return Credentials{}, err
			}

			creds.Value = ""
			return creds, nil
		} else if len(DBUser.Credentials.Key) == 0 && len(DBUser.Credentials.Value) == 0 {
			DBUser.Credentials.Key = PassHash(creds.Key)
			DBUser.Credentials.Value = Hash(creds.Value)
			if err := UpdateItem(db, UserTable, DBUser.Credentials.Value, DBUser); err != nil {
				return Credentials{}, err
			}
			return creds, nil
		}
	}

	isNewUser := true
	if ok && len(DBUser.Credentials.Value) > 0 {
		// UUID already exists
		if len(u.Credentials.Key) > 0 && IsValidCredentials(u.Credentials.Value) {
			// If client passes current details they are asking for new Credentials.
			// Verify the Credentials passed are valid
			if u.Verify(db) {
				isNewUser = false
			} else {
				return Credentials{}, errors.New("Unable to create new credentials.")
			}
		}
	}

	u.Credentials.Value = Hash(creds.Value)
	u.Credentials.Key = PassHash(creds.Key)

	if isNewUser {
		// create new user
		if err := AddItem(db, UserTable, u); err != nil {
			return Credentials{}, err
		}
	} else {
		// update user
		if err := UpdateItem(db, UserTable, DBUser.Credentials.Value, u); err != nil {
			return Credentials{}, err
		}
	}
	return creds, nil
}

// Verify verifies a u User s credentials
func (u User) Verify(db *dynamo.DB) bool {
	result, err := GetItem(db, UserTable, "credentials", u.Credentials.Value)
	if err != nil {
		return false
	}
	user := result.(User)
	isValidKey := VerifyPassHash(user.Credentials.Key, u.Credentials.Key)
	isValidUUID := user.UUID == Hash(u.UUID)
	return isValidKey && isValidUUID
}
