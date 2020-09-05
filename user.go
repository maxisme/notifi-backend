package main

import (
	"database/sql"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"

	"github.com/go-errors/errors"
	"github.com/maxisme/notifi-backend/crypt"
	tdb "github.com/maxisme/notifi-backend/tracer/db"
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
	Value credentials `json:"credentials"`
	Key   string      `json:"credential_key"`
}

const (
	credentialLen    = 25
	credentialKeyLen = 100
)

// Store stores or updates u User with new Credentials depending on whether the user passes current Credentials
// in the u User struct. TODO badly structured separate update and store

func (u User) Store(r *http.Request, db *sql.DB) (Credentials, error) {
	// create new credentials
	creds := Credentials{
		crypt.RandomString(credentialLen),
		crypt.RandomString(credentialKeyLen),
	}

	var DBUser User
	_ = DBUser.GetWithUUID(r, db, u.UUID) // doesn't matter if error just means there is no previous user with UUID
	if len(DBUser.UUID) > 0 {
		if len(DBUser.Credentials.Key) == 0 && len(DBUser.Credentials.Value) > 0 {
			Log(r, log.InfoLevel, fmt.Sprintf("Credential reset for: %s", crypt.Hash(u.UUID)))

			// language=MySQL
			query := "UPDATE users SET credential_key = ? WHERE UUID = ?"
			_, err := tdb.Exec(r, db, query, crypt.PassHash(creds.Key), crypt.Hash(u.UUID))
			if err != nil {
				Log(r, log.ErrorLevel, err.Error())
				return Credentials{}, err
			}
			creds.Value = ""
			return creds, nil
		} else if len(DBUser.Credentials.Key) == 0 && len(DBUser.Credentials.Value) == 0 {
			Log(r, log.InfoLevel, fmt.Sprintf("Account reset for: %s", crypt.Hash(u.UUID)))

			// language=MySQL
			query := "UPDATE users SET credential_key = ?, credentials = ? WHERE UUID = ?"
			_, err := tdb.Exec(r, db, query, crypt.PassHash(creds.Key), crypt.Hash(creds.Value), crypt.Hash(u.UUID))
			if err != nil {
				Log(r, log.ErrorLevel, err.Error())
				return Credentials{}, err
			}
			return creds, nil
		}
	}

	isNewUser := true
	if len(DBUser.Credentials.Value) > 0 {
		// UUID already exists
		if len(u.Credentials.Key) > 0 && len(u.Credentials.Value) > 0 {
			// if client passes current details they are asking for new Credentials

			// verify the Credentials passed are valid
			if u.Verify(r, db) {
				isNewUser = false
			} else {
				Log(r, log.WarnLevel, fmt.Sprintf("Client passed credentials that were invalid"))
				return Credentials{}, errors.New("Unable to create new credentials.")
			}
		}
	}

	// update users Credentials
	query := ""
	if isNewUser {
		// create new user
		// language=MySQL
		query = `
		INSERT INTO users (credentials, credential_key, UUID) 
		VALUES (?, ?, ?)`
	} else {
		// language=MySQL
		query = `
		UPDATE users SET credentials = ?, credential_key = ?
		WHERE UUID = ?`
	}

	_, err := tdb.Exec(r, db, query, crypt.Hash(creds.Value), crypt.PassHash(creds.Key), crypt.Hash(u.UUID))
	if err != nil {
		return Credentials{}, err
	}
	return creds, nil
}

// GetWithUUID will return user params based on a UUID
func (u *User) GetWithUUID(r *http.Request, db *sql.DB, UUID string) error {
	// language=MySQL
	row := tdb.QueryRow(r, db, `
	SELECT UUID, credentials, credential_key 
	FROM users
	WHERE UUID = ?
	`, crypt.Hash(UUID))
	return row.Scan(&u.UUID, &u.Credentials.Value, &u.Credentials.Key)
}

// Get will return user params based on Credentials
func (u *User) Get(r *http.Request, db *sql.DB, credentials string) error {
	// language=MySQL
	var row = tdb.QueryRow(r, db, `
	SELECT UUID, credentials, credential_key 
	FROM users
	WHERE credentials = ?
	`, crypt.Hash(credentials))
	return row.Scan(&u.UUID, &u.Credentials.Value, &u.Credentials.Key)
}

// Verify verifies a u User s credentials
func (u User) Verify(r *http.Request, db *sql.DB) bool {
	var DBUser User
	err := DBUser.Get(r, db, fmt.Sprint(u.Credentials.Value))
	if err != nil {
		return false
	}

	isValidKey := crypt.VerifyPassHash(DBUser.Credentials.Key, u.Credentials.Key)
	isValidUUID := DBUser.UUID == crypt.Hash(u.UUID)
	if isValidKey && isValidUUID {
		return true
	}
	return false
}

// StoreLogin stores the current timestamp that the user has connected to the web socket as well as the app version
// the client is using and the public serverKey to encrypt messages on the Server with
func (u User) StoreLogin(r *http.Request, db *sql.DB) error {
	// language=MySQL
	return UpdateErr(tdb.Exec(r, db, `UPDATE users
	SET last_login = NOW(), app_version = ?, is_connected = 1
	WHERE credentials = ? AND UUID = ?`, u.AppVersion, crypt.Hash(u.Credentials.Value), crypt.Hash(u.UUID)))
}

// CloseLogin marks a user as no longer connected to web socket in db
func (u User) CloseLogin(r *http.Request, db *sql.DB) error {
	// language=MySQL
	return UpdateErr(tdb.Exec(r, db, `UPDATE users
	SET is_connected = 0
	WHERE credentials = ? AND UUID = ?`, crypt.Hash(u.Credentials.Value), crypt.Hash(u.UUID)))
}
