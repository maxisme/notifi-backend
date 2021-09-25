package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func HandleCode(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// create PostUser struct
	PostUser := User{
		UUID:          r.Form.Get("UUID"),
		FirebaseToken: r.Form.Get("firebase_token"),

		// when asking for new Credentials
		CredentialsKey: r.Form.Get("current_credential_key"),
		Credentials:    r.Form.Get("current_credentials"),
		Created:        time.Now(),
	}

	if !IsValidUUID(PostUser.UUID) {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	PostUser.UUID = Hash(PostUser.UUID)

	db, err := GetDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	creds, err := PostUser.Store(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c, err := json.Marshal(creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(c)
}
