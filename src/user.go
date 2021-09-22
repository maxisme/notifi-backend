package main

import (
	"github.com/go-errors/errors"
	"github.com/guregu/dynamo"
	"time"
)

// User structure
type User struct {
	AppVersion      string    `dynamo:"app_version"`
	Created         time.Time `dynamo:"created_dttm"`
	Credentials     string    `dynamo:"credentials,hash"`
	CredentialsKey  string    `dynamo:"credential_key"`
	Device          string    `dynamo:"device_info"`
	FirebaseToken   string    `dynamo:"firebase_token,allowempty"`
	LastLogin       time.Time `dynamo:"last_login_dttm"`
	NotificationCnt int       `dynamo:"notification_cnt"`
	UUID            string    `dynamo:"device_uuid,hash"`
}

type Ws struct {
	ConnectionID string `dynamo:"connection_id,hash"`
	UUID         string `dynamo:"device_uuid,hash"`
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
const UserTable = "user"
const WsTable = "ws"

// Store stores or updates u User with new Credentials depending on whether the user passes current Credentials
// in the u User struct. TODO badly structured separate update and store

func (u User) Store(db *dynamo.DB) (Credentials, error) {
	// create new credentials
	creds := Credentials{
		RandomString(credentialLen),
		RandomString(credentialKeyLen),
	}

	var DBUser User
	_ = db.Table(UserTable).Get("device_uuid", u.UUID).One(&DBUser)
	if len(DBUser.UUID) > 0 {
		if len(DBUser.CredentialsKey) == 0 && len(DBUser.Credentials) > 0 {
			DBUser.CredentialsKey = PassHash(creds.Key)
			if err := UpdateItem(db, UserTable, DBUser.Credentials, DBUser); err != nil {
				return Credentials{}, err
			}
			creds.Value = ""
			return creds, nil
		} else if len(DBUser.CredentialsKey) == 0 && len(DBUser.Credentials) == 0 {
			DBUser.CredentialsKey = PassHash(creds.Key)
			DBUser.Credentials = Hash(creds.Value)
			if err := UpdateItem(db, UserTable, DBUser.Credentials, DBUser); err != nil {
				return Credentials{}, err
			}
			return creds, nil
		}
	}

	isNewUser := true
	if len(DBUser.Credentials) > 0 {
		// UUID already exists
		if len(u.CredentialsKey) > 0 && IsValidCredentials(u.Credentials) {
			// If client passes current details they are asking for new Credentials.
			// Verify the Credentials passed are valid
			if u.Verify(db) {
				isNewUser = false
			} else {
				return Credentials{}, errors.New("Unable to create new credentials.")
			}
		}
	}

	if isNewUser && len(DBUser.UUID) > 0 {
		return Credentials{}, errors.New("UUID already used")
	}

	u.Credentials = Hash(creds.Value)
	u.CredentialsKey = PassHash(creds.Key)

	if isNewUser {
		// create new user
		if err := AddItem(db, UserTable, u); err != nil {
			return Credentials{}, err
		}
	} else {
		// update user
		if err := UpdateItem(db, UserTable, DBUser.Credentials, u); err != nil {
			return Credentials{}, err
		}
	}
	return creds, nil
}

// Verify verifies a u User s credentials
func (u User) Verify(db *dynamo.DB) bool {
	var user User
	err := db.Table(UserTable).Get("device_uuid", u.UUID).One(&user)
	if err != nil {
		return false
	}
	isValidKey := VerifyPassHash(user.CredentialsKey, u.CredentialsKey)
	isValidUUID := user.UUID == Hash(u.UUID)
	return isValidKey && isValidUUID
}
