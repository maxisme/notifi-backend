package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

const RequestNewUserCode = 551

func HandleConnect(ctx context.Context, r events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	user := User{
		Credentials:    r.Headers["Credentials"],
		CredentialsKey: r.Headers["Key"],
		UUID:           r.Headers["Uuid"],
		AppVersion:     r.Headers["Version"],
		LastLogin:      time.Now(),
	}

	firebaseToken, ok := r.Headers["Firebase-Token"]
	if ok {
		user.FirebaseToken = firebaseToken
	}

	// validate inputs
	if !IsValidUUID(user.UUID) {
		return WriteError(errors.New("Invalid UUID"), http.StatusBadRequest)
	} else if !IsValidVersion(user.AppVersion) {
		return WriteError(fmt.Errorf("Invalid Version %v", user.AppVersion), http.StatusBadRequest)
	} else if !IsValidCredentials(user.Credentials) {
		return WriteError(fmt.Errorf("Invalid Credentials"), http.StatusForbidden)
	}

	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	var DBUser User
	err = db.Table(UserTable).Get("device_uuid", Hash(user.UUID)).One(&DBUser)
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}
	var errorCode, errorMsg = 0, ""
	if len(DBUser.CredentialsKey) == 0 {
		errorCode = RequestNewUserCode
		if len(DBUser.Credentials) == 0 {
			errorMsg = "No credentials or key for: " + user.UUID
		} else {
			errorMsg = "No credential key for: " + user.UUID
		}
	} else if !user.Verify(db) {
		errorMsg = "Forbidden"
		errorCode = http.StatusForbidden
	}

	if errorCode != 0 {
		return WriteError(fmt.Errorf(errorMsg), errorCode)
	}

	// store user info in db
	err = db.Table(UserTable).Update("device_uuid", Hash(user.UUID)).Run()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	sesh := NewAPIGatewaySession()
	if err := SendWsMessage(sesh, r.RequestContext.ConnectionID, []byte(".")); err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	// send all stored notifications from db
	if err := SendStoredMessages(db, Hash(user.Credentials), r.RequestContext.ConnectionID); err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return WriteEmptySuccess()
}
