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
	FirebaseToken   string
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

			// language=PostgreSQL
			query := "UPDATE users SET credential_key = $1 WHERE UUID = $2"
			_, err := db.Exec(query, crypt.PassHash(creds.Key), crypt.Hash(u.UUID))
			if err != nil {
				Log(r, log.ErrorLevel, err.Error())
				return Credentials{}, err
			}
			creds.Value = ""
			return creds, nil
		} else if len(DBUser.Credentials.Key) == 0 && len(DBUser.Credentials.Value) == 0 {
			Log(r, log.InfoLevel, fmt.Sprintf("Account reset for: %s", crypt.Hash(u.UUID)))

			// language=PostgreSQL
			query := "UPDATE users SET credential_key = $1, credentials = $2 WHERE UUID = $3"
			_, err := db.Exec(query, crypt.PassHash(creds.Key), crypt.Hash(creds.Value), crypt.Hash(u.UUID))
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
		if len(u.Credentials.Key) > 0 && IsValidCredentials(u.Credentials.Value) {
			// If client passes current details they are asking for new Credentials.
			// Verify the Credentials passed are valid
			if u.Verify(r, db) {
				isNewUser = false
			} else {
				Log(r, log.WarnLevel, fmt.Sprintf("Client passed credentials that were invalid: %s", crypt.Hash(u.Credentials.Value)))
				return Credentials{}, errors.New("Unable to create new credentials.")
			}
		}
	}

	// update users Credentials
	query := ""
	if isNewUser {
		// create new user
		// language=PostgreSQL
		query = "INSERT INTO users (credentials, credential_key, firebase_token, UUID) VALUES ($1, $2, $3, $4)"
	} else {
		// update user
		// language=PostgreSQL
		query = "UPDATE users SET credentials = $1, credential_key = $2, firebase_token = $3 WHERE UUID = $4"
	}

	_, err := tdb.Exec(r, db, query, crypt.Hash(creds.Value), crypt.PassHash(creds.Key), u.FirebaseToken, crypt.Hash(u.UUID))
	if err != nil {
		return Credentials{}, err
	}
	return creds, nil
}

// GetWithUUID will return user params based on a UUID
func (u *User) GetWithUUID(r *http.Request, db *sql.DB, UUID string) error {
	// language=PostgreSQL
	row := tdb.QueryRow(r, db, `
	SELECT UUID, credentials, credential_key 
	FROM users
	WHERE UUID = $1
	`, crypt.Hash(UUID))
	return row.Scan(&u.UUID, &u.Credentials.Value, &u.Credentials.Key)
}

// Get will return user params based on Credentials
func (u *User) Get(r *http.Request, db *sql.DB, credentials string) error {
	var firebaseToken sql.NullString
	// language=PostgreSQL
	var row = tdb.QueryRow(r, db, `
	SELECT UUID, credentials, credential_key, notification_cnt, firebase_token
	FROM users
	WHERE credentials = $1
	`, crypt.Hash(credentials))
	if err := row.Scan(&u.UUID, &u.Credentials.Value, &u.Credentials.Key, &u.NotificationCnt, &firebaseToken); err != nil {
		return err
	}
	u.FirebaseToken = firebaseToken.String
	return nil
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
	// language=PostgreSQL
	return UpdateErr(tdb.Exec(r, db, `UPDATE users
	SET last_login = NOW(), app_version = $1, is_connected = true, firebase_token = $2
	WHERE credentials = $3 AND UUID = $4`, u.AppVersion, u.FirebaseToken, crypt.Hash(u.Credentials.Value), crypt.Hash(u.UUID)))
}

// CloseLogin marks a user as no longer connected to web socket in db
func (u User) CloseLogin(r *http.Request, db *sql.DB) error {
	// language=PostgreSQL
	return UpdateErr(tdb.Exec(r, db, `UPDATE users
	SET is_connected = false
	WHERE credentials = $1 AND UUID = $2`, crypt.Hash(u.Credentials.Value), crypt.Hash(u.UUID)))
}
