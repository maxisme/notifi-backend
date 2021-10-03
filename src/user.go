package main

import (
	"github.com/go-errors/errors"
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
	Device          string    `dynamo:"device_info"`
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
func (u User) Store(db *dynamo.DB) (Credentials, error) {
	// create new credentials
	credentials := Credentials{
		RandomString(credentialLen),
		RandomString(credentialKeyLen),
	}

	var StoredUser User
	_ = db.Table(UserTable).Get("device_uuid", u.UUID).One(&StoredUser)
	if len(StoredUser.UUID) > 0 {
		if len(StoredUser.CredentialsKey) == 0 && len(StoredUser.Credentials) > 0 {
			StoredUser.CredentialsKey = PassHash(credentials.Key)
			if err := db.Table(UserTable).Put(StoredUser).Run(); err != nil {
				return Credentials{}, err
			}
			credentials.Value = ""
			return credentials, nil
		} else if len(StoredUser.CredentialsKey) == 0 && len(StoredUser.Credentials) == 0 {
			StoredUser.CredentialsKey = PassHash(credentials.Key)
			StoredUser.Credentials = Hash(credentials.Value)
			if err := db.Table(UserTable).Put(StoredUser).Run(); err != nil {
				return Credentials{}, err
			}
			return credentials, nil
		}
	}

	isNewUser := true
	if len(StoredUser.Credentials) > 0 {
		// UUID already exists
		if len(u.CredentialsKey) > 0 && IsValidCredentials(u.Credentials) {
			// If client passes current details they are asking for new Credentials.
			// Verify the Credentials passed are valid
			if u.Verify(StoredUser) {
				isNewUser = false
			} else {
				return Credentials{}, errors.New("Unable to create new credentials.")
			}
		}
	}

	if isNewUser && len(StoredUser.UUID) > 0 {
		return Credentials{}, errors.New("UUID already used")
	}

	u.Credentials = Hash(credentials.Value)
	u.CredentialsKey = PassHash(credentials.Key)

	// create or update new user
	if err := db.Table(UserTable).Put(u).Run(); err != nil {
		return Credentials{}, err
	}
	return credentials, nil
}

// Verify verifies a u User s credentials
func (u User) Verify(user User) bool {
	isValidKey := VerifyPassHash(user.CredentialsKey, u.CredentialsKey)
	isValidUUID := user.UUID == Hash(u.UUID)
	return isValidKey && isValidUUID
}
