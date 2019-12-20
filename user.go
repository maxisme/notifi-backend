package main

import (
	"database/sql"
	"github.com/go-errors/errors"
	"log"
)

type User struct {
	ID              int
	Created         string
	Credentials     Credentials
	LastLogin       string
	isConnected     string
	AppVersion      string
	NotificationCnt string
	UUID            string
}

type Credentials struct {
	Value string `json:"credentials"`
	Key   string `json:"key"`
}

// create user or update user with new credentials depending on whether the user passes current credentials
// in the User struct.
func CreateUser(db *sql.DB, u User) (Credentials, error) {
	// create new credentials
	creds := Credentials{
		RandomString(25),
		RandomString(100),
	}

	dbu := FetchUserCredentialsFromUUID(db, u.UUID)
	if len(dbu.Credentials.Key) == 0 && len(dbu.Credentials.Value) > 0 {
		// update users credential key if not set in db
		query := `UPDATE users SET credential_key = ?
		WHERE UUID = ?`
		_, err := db.Exec(query, PassHash(creds.Key), Hash(u.UUID))
		if err != nil {
			return Credentials{}, err
		}
		creds.Value = ""
		return creds, nil
	}

	isnewuser := true
	if len(dbu.Credentials.Value) > 0 {
		// UUID already exists
		if len(u.Credentials.Key) > 0 && len(u.Credentials.Value) > 0 {
			// if client passes current details they are asking for new credentials

			// verify the credentials passed are valid
			if VerifyUser(db, u) {
				isnewuser = false
			} else {
				log.Print("Lied about credentials ")
				return Credentials{}, errors.New("Unable to create new credentials.")
			}
		}
	}

	// update users credentials
	query := ""
	if isnewuser {
		// create new user
		query = `
		INSERT INTO users (credentials, credential_key, UUID) 
		VALUES (?, ?, ?)`
	} else {
		query = `
		UPDATE users SET credentials = ?, credential_key = ?
		WHERE UUID = ?`
	}

	_, err := db.Exec(query, Hash(creds.Value), PassHash(creds.Key), Hash(u.UUID))
	if err != nil {
		Handle(err)
		return Credentials{}, err
	}
	return creds, nil
}

func FetchUser(db *sql.DB, credentials string) User {
	var u User
	_ = db.QueryRow(`
	SELECT credential_key, UUID
	FROM users 
	WHERE credentials = ?`, Hash(credentials)).Scan(&u.Credentials.Key, &u.UUID)
	return u
}

func FetchUserCredentialsFromUUID(db *sql.DB, UUID string) User {
	var u User
	_ = db.QueryRow(`
	SELECT credential_key, credentials
	FROM users 
	WHERE UUID = ?`, Hash(UUID)).Scan(&u.Credentials.Key, &u.Credentials.Value)
	return u
}

func VerifyUser(db *sql.DB, u User) bool {
	storeduser := FetchUser(db, u.Credentials.Value)

	valid_key := VerifyPassHash(storeduser.Credentials.Key, u.Credentials.Key)
	valid_UUID := storeduser.UUID == Hash(u.UUID)
	if valid_key && valid_UUID {
		return true
	}
	return false
}

// stores the current timestamp that the user has connected to the wss
// as well as the app version the client is using
func SetLastLogin(db *sql.DB, u User) error {
	_, err := db.Exec(`UPDATE users
	SET last_login = NOW(), app_version = ?, is_connected = 1
	WHERE credentials = ? AND UUID = ?`, u.AppVersion, Hash(u.Credentials.Value), Hash(u.UUID))
	return err
}

func CloseConnection(db *sql.DB, u User) error {
	_, err := db.Exec(`UPDATE users
	SET is_connected = 0
	WHERE credentials = ? AND UUID = ?`, Hash(u.Credentials.Value), Hash(u.UUID))
	return err
}
