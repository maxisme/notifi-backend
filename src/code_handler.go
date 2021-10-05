package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func HandleCode(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Sec-Key") != os.Getenv("SERVER_KEY") {
		return
	}

	err := r.ParseForm()
	if err != nil {
		WriteHttpError(w, err, http.StatusBadRequest)
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
		WriteHttpError(w, fmt.Errorf("Invalid UUID"), http.StatusBadRequest)
		return
	}

	db, err := GetDB()
	if err != nil {
		WriteHttpError(w, err, http.StatusInternalServerError)
		return
	}

	creds, err := PostUser.Store(db)
	if err != nil {
		WriteHttpError(w, err, http.StatusInternalServerError)
		return
	}

	c, err := json.Marshal(creds)
	if err != nil {
		WriteHttpError(w, err, http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(c)
}
