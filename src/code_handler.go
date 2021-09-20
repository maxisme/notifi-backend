package main

import (
	"encoding/json"
	"net/http"
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

		// if asking for new Credentials
		Credentials: Credentials{
			Value: r.Form.Get("current_credentials"),
			Key:   r.Form.Get("current_credential_key"),
		},
	}

	if !IsValidUUID(PostUser.UUID) {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

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
