package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

const RequestNewUserCode = 551

func HandleConnect(_ context.Context, r events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	if r.Headers["sec-key"] != os.Getenv("SERVER_KEY") {
		return WriteError(fmt.Errorf("Invalid server key"), http.StatusForbidden)
	}

	user := User{
		Credentials:    r.Headers["credentials"],
		CredentialsKey: r.Headers["key"],
		UUID:           r.Headers["uuid"],
	}

	// validate inputs
	if !IsValidUUID(user.UUID) {
		return WriteError(fmt.Errorf("Invalid UUID '%s'", user.UUID), http.StatusBadRequest)
	} else if !IsValidVersion(r.Headers["version"]) {
		return WriteError(fmt.Errorf("Invalid Version %v", r.Headers["version"]), http.StatusBadRequest)
	} else if !IsValidCredentials(user.Credentials) {
		return WriteError(fmt.Errorf("Invalid Credentials"), http.StatusForbidden)
	}

	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	var StoredUser User
	err = db.Table(UserTable).Get("device_uuid", Hash(user.UUID)).One(&StoredUser)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"uuid": Hash(user.UUID),
			"err":  err.Error(),
		}).Error("Trying to connect without credentials...")
		return WriteError(err, http.StatusInternalServerError)
	}

	var errorCode, errorMsg = 0, ""
	if len(StoredUser.CredentialsKey) == 0 {
		errorCode = RequestNewUserCode
		if len(StoredUser.Credentials) == 0 {
			errorMsg = "No credentials or key for: " + user.UUID
		} else {
			errorMsg = "No credential key for: " + user.UUID
		}
	} else if !user.Verify(StoredUser) {
		errorMsg = "Forbidden"
		errorCode = http.StatusForbidden
	} else if len(StoredUser.ConnectionID) > 0 {
		if err := CloseConnection(StoredUser.ConnectionID); err != nil {
			logrus.WithFields(logrus.Fields{
				"connection_id": StoredUser.ConnectionID,
				"uuid":          Hash(user.UUID),
				"err":           err.Error(),
			}).Error("problem closing already open ws connection")
		}
	}

	if errorCode != 0 {
		return WriteError(errors.New(errorMsg), errorCode)
	}

	StoredUser.AppVersion = r.Headers["version"]
	if firebaseToken, ok := r.Headers["firebase-token"]; ok {
		StoredUser.FirebaseToken = firebaseToken
	}
	if operatingSystem, ok := r.Headers["os"]; ok {
		StoredUser.OS = operatingSystem
	}
	StoredUser.LastLogin = time.Now()
	StoredUser.ConnectionID = r.RequestContext.ConnectionID

	// update user info in db
	err = db.Table(UserTable).Put(StoredUser).Run()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return WriteEmptySuccess()
}
