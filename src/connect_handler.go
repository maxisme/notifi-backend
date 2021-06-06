package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const RequestNewUserCode = 551

func main() {
	lambda.Start(HandleConnect)
}

func HandleConnect(ctx context.Context, r events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	if r.HTTPMethod != "GET" {
		return WriteError(errors.New("Method not allowed"), http.StatusBadRequest)
	}

	user := User{
		Credentials: Credentials{
			Value: r.Headers["Credentials"],
			Key:   r.Headers["Key"],
		},
		UUID:         r.Headers["Uuid"],
		AppVersion:   r.Headers["Version"],
		ConnectionID: r.RequestContext.ConnectionID,
		LastLogin:    time.Now(),
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
	} else if !IsValidCredentials(user.Credentials.Value) {
		return WriteError(fmt.Errorf("Invalid Credentials"), http.StatusForbidden)
	}

	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	result, _ := GetItem(db, UserTable, "UUID", user.UUID)
	DBUser := result.(User)
	var errorCode = 0
	var errorMsg = ""
	if len(DBUser.Credentials.Key) == 0 {
		errorCode = RequestNewUserCode
		if len(DBUser.Credentials.Value) == 0 {
			errorMsg = "No credentials or key for: " + user.UUID
		} else {
			errorMsg = "No credential key for: " + user.UUID
		}
	} else if !user.Verify(db) {
		errorCode = http.StatusForbidden
	}

	if errorCode != 0 {
		return WriteError(fmt.Errorf(errorMsg), errorCode)
	}

	// store user info in db
	if err := UpdateItem(db, UserTable, user.Credentials.Value, user); err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	sesh := NewAPIGatewaySession()
	if err := SendWsMessage(sesh, r.RequestContext.ConnectionID, []byte(".")); err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	// send all stored notifications from db

	if err := SendStoredMessages(db, Hash(user.Credentials.Value), r.RequestContext.ConnectionID); err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return WriteSuccess()
}
