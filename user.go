package main

import (
	"database/sql"
	"net/http"

	"github.com/go-errors/errors"
	"github.com/maxisme/notifi-backend/crypt"
)

// User structure
type User struct {
	ID              int
	Created         string
	Credentials     Credentials
	LastLogin       string
	AppVersion      string
	NotificationCnt string
	UUID            string
}

// Credentials structure
type credentials = string
type Credentials struct {
	Value   credentials `json:"credentials"`
	UUIDKey string      `json:"UUIDKey"`
}

const (
	credentialLen = 25
	UUIDKeyLen    = 100
)

// Store stores or updates u User with new Credentials depending on whether the user passes current Credentials
// in the u User struct. TODO badly structured separate update and store

func (u User) Store(r *http.Request, db *sql.DB) (Credentials, error) {
	// create new credentials
	creds := Credentials{
		crypt.RandomString(credentialLen),
		crypt.RandomString(UUIDKeyLen),
	}

	var DBUser User
	_ = DBUser.GetWithUUID(db, u.UUID) // doesn't matter if error just means there is no previous user with UUID
	if len(DBUser.UUID) > 0 {
		LogInfo(r, DBUser.UUID+" has an account already")

		if len(DBUser.Credentials.UUIDKey) == 0 && len(DBUser.Credentials.Value) > 0 {
			LogInfo(r, "Credential key reset for: "+crypt.Hash(u.UUID))

			query := "UPDATE users SET credential_key = ? WHERE UUID = ?"
			_, err := db.Exec(query, crypt.PassHash(creds.UUIDKey), crypt.Hash(u.UUID))
			if err != nil {
				Handle(r, err)
				return Credentials{}, err
			}
			creds.Value = ""
			return creds, nil
		} else if len(DBUser.Credentials.UUIDKey) == 0 && len(DBUser.Credentials.Value) == 0 {
			LogInfo(r, "Account reset for: "+crypt.Hash(u.UUID))

			query := "UPDATE users SET credential_key = ?, credentials = ? WHERE UUID = ?"
			_, err := db.Exec(query, crypt.PassHash(creds.UUIDKey), crypt.Hash(creds.Value), crypt.Hash(u.UUID))
			if err != nil {
				Handle(r, err)
				return Credentials{}, err
			}
			return creds, nil
		}
	}

	isNewUser := true
	if len(DBUser.Credentials.Value) > 0 {
		// UUID already exists
		if len(u.Credentials.UUIDKey) > 0 && len(u.Credentials.Value) > 0 {
			// if client passes current details they are asking for new Credentials

			// verify the Credentials passed are valid
			if u.Verify(r, db) {
				isNewUser = false
			} else {
				LogInfo(r, "Client lied about credentials") // TODO better logging
				return Credentials{}, errors.New("Unable to create new credentials.")
			}
		}
	}

	// update users Credentials
	query := ""
	if isNewUser {
		// create new user
		query = `
		INSERT INTO users (credentials, credential_key, UUID) 
		VALUES (?, ?, ?)`
	} else {
		query = `
		UPDATE users SET credentials = ?, credential_key = ?
		WHERE UUID = ?`
	}

	_, err := db.Exec(query, crypt.Hash(creds.Value), crypt.PassHash(creds.UUIDKey), crypt.Hash(u.UUID))
	if err != nil {
		return Credentials{}, err
	}
	return creds, nil
}

// GetWithUUID will return user params based on a UUID
func (u *User) GetWithUUID(db *sql.DB, UUID string) error {
	row := db.QueryRow(`
	SELECT UUID, credentials, credential_key 
	FROM users
	WHERE UUID = ?
	`, crypt.Hash(UUID))
	return row.Scan(&u.UUID, &u.Credentials.Value, &u.Credentials.UUIDKey)
}

// Get will return user params based on Credentials
func (u *User) Get(db *sql.DB, credentials string) error {
	var row = db.QueryRow(`
	SELECT UUID, credentials, credential_key 
	FROM users
	WHERE credentials = ?
	`, crypt.Hash(credentials))
	return row.Scan(&u.UUID, &u.Credentials.Value, &u.Credentials.UUIDKey)
}

// Verify verifies a u User s credentials
func (u User) Verify(r *http.Request, db *sql.DB) bool {
	var DBUser User
	err := DBUser.Get(db, string(u.Credentials.Value))
	if err != nil {
		LogInfo(r, "No such credentials in db: "+u.Credentials.Value)
		return false
	}

	isValidKey := crypt.VerifyPassHash(DBUser.Credentials.UUIDKey, u.Credentials.UUIDKey)
	isValidUUID := DBUser.UUID == crypt.Hash(u.UUID)
	if isValidKey && isValidUUID {
		return true
	}
	return false
}

// StoreLogin stores the current timestamp that the user has connected to the web socket as well as the app version
// the client is using and the public key to encrypt messages on the Server with
func (u User) StoreLogin(db *sql.DB) error {
	return UpdateErr(db.Exec(`UPDATE users
	SET last_login = NOW(), app_version = ?, is_connected = 1
	WHERE credentials = ? AND UUID = ?`, u.AppVersion, crypt.Hash(u.Credentials.Value), crypt.Hash(u.UUID)))
}

// CloseLogin marks a user as no longer connected to web socket in db
func (u User) CloseLogin(db *sql.DB) error {
	return UpdateErr(db.Exec(`UPDATE users
	SET is_connected = 0
	WHERE credentials = ? AND UUID = ?`, crypt.Hash(u.Credentials.Value), crypt.Hash(u.UUID)))
}
