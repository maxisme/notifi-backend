package main

import (
	"database/sql"
	"github.com/go-errors/errors"
	"github.com/maxisme/notifi-backend/crypt"
	"log"
)

type User struct {
	ID              int
	Created         string
	Credentials     Credentials
	LastLogin       string
	AppVersion      string
	NotificationCnt string
	UUID            string
}

type Credentials struct {
	Value string `json:"credentials"`
	Key   string `json:"key"`
}

var (
	CREDENTIALLEN    = 25
	CREDENTIALKEYLEN = 100
)

// create user or update user with new credentials depending on whether the user passes current credentials
// in the u User struct.
func (u User) Store(db *sql.DB) (Credentials, error) {
	// create new credentials
	creds := Credentials{
		crypt.RandomString(CREDENTIALLEN),
		crypt.RandomString(CREDENTIALKEYLEN),
	}

	var DBUser User
	_ = DBUser.GetWithUUID(db, u.UUID) // doesn't matter if error just means there is no previous user with UUID
	if len(DBUser.UUID) > 0 {
		log.Println(DBUser.UUID + " has an account already")

		if len(DBUser.Credentials.Key) == 0 && len(DBUser.Credentials.Value) > 0 {
			log.Println("Credential key reset for: " + crypt.Hash(u.UUID))

			query := "UPDATE users SET credential_key = ? WHERE UUID = ?"
			_, err := db.Exec(query, crypt.PassHash(creds.Key), crypt.Hash(u.UUID))
			if err != nil {
				Handle(err)
				return Credentials{}, err
			}
			creds.Value = ""
			return creds, nil
		} else if len(DBUser.Credentials.Key) == 0 && len(DBUser.Credentials.Value) == 0 {
			log.Println("Account reset for: " + crypt.Hash(u.UUID))

			query := "UPDATE users SET credential_key = ?, credentials = ? WHERE UUID = ?"
			_, err := db.Exec(query, crypt.PassHash(creds.Key), crypt.Hash(creds.Value), crypt.Hash(u.UUID))
			if err != nil {
				Handle(err)
				return Credentials{}, err
			}
			return creds, nil
		}
	}

	isNewUser := true
	if len(DBUser.Credentials.Value) > 0 {
		// UUID already exists
		if len(u.Credentials.Key) > 0 && len(u.Credentials.Value) > 0 {
			// if client passes current details they are asking for new credentials

			// verify the credentials passed are valid
			if u.Verify(db) {
				isNewUser = false
			} else {
				log.Println("Lied about credentials") // TODO better logging
				return Credentials{}, errors.New("Unable to create new credentials.")
			}
		}
	}

	// update users credentials
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

	_, err := db.Exec(query, crypt.Hash(creds.Value), crypt.PassHash(creds.Key), crypt.Hash(u.UUID))
	if err != nil {
		return Credentials{}, err
	}
	return creds, nil
}

func (u *User) GetWithUUID(db *sql.DB, UUID string) error {
	row := db.QueryRow(`
	SELECT UUID, credentials, credential_key 
	FROM users
	WHERE UUID = ?
	`, crypt.Hash(UUID))
	return row.Scan(&u.UUID, &u.Credentials.Value, &u.Credentials.Key)
}

func (u *User) Get(db *sql.DB, credentials string) error {
	row := db.QueryRow(`
	SELECT UUID, credentials, credential_key 
	FROM users
	WHERE credentials = ?
	`, crypt.Hash(credentials))
	return row.Scan(&u.UUID, &u.Credentials.Value, &u.Credentials.Key)
}

func (u User) Verify(db *sql.DB) bool {
	var DBUser User
	err := DBUser.Get(db, u.Credentials.Value)
	if err != nil {
		log.Println("No such credentials in db: " + u.Credentials.Value)
		return false
	}

	valid_key := crypt.VerifyPassHash(DBUser.Credentials.Key, u.Credentials.Key)
	valid_UUID := DBUser.UUID == crypt.Hash(u.UUID)
	if valid_key && valid_UUID {
		return true
	}
	return false
}

// stores the current timestamp that the user has connected to the wss
// as well as the app version the client is using
// and the public key to encrypt messages on the Server with
func (u User) StoreLogin(db *sql.DB) error {
	_, err := db.Exec(`UPDATE users
	SET last_login = NOW(), app_version = ?, is_connected = 1
	WHERE credentials = ? AND UUID = ?`, u.AppVersion, crypt.Hash(u.Credentials.Value), crypt.Hash(u.UUID))
	return err
}

func (u User) CloseLogin(db *sql.DB) error {
	_, err := db.Exec(`UPDATE users
	SET is_connected = 0
	WHERE credentials = ? AND UUID = ?`, crypt.Hash(u.Credentials.Value), crypt.Hash(u.UUID))
	return err
}
