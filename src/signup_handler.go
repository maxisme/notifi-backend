package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"net/http"
)

func main() {
	lambda.Start(HandleSignup)
}

func HandleSignup(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if r.HTTPMethod != "POST" {
		return WriteError(fmt.Errorf("Method not allowed"), http.StatusBadRequest)
	}

	// create PostUser struct
	PostUser := User{
		UUID: r.StageVariables["UUID"],

		// if asking for new Credentials
		Credentials: Credentials{
			Value: r.StageVariables["current_credentials"],
			Key:   r.StageVariables["current_credential_key"],
		},
	}

	firebaseToken, ok := r.StageVariables["current_credential_key"]
	if ok {
		PostUser.FirebaseToken = firebaseToken
	}

	if !IsValidUUID(PostUser.UUID) {
		return WriteError(fmt.Errorf("Invalid UUID"), http.StatusBadRequest)
	}

	db, err := GetDB()
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	creds, err := PostUser.Store(db)
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	c, err := json.Marshal(creds)
	if err != nil {
		return WriteError(err, http.StatusInternalServerError)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(c),
	}, nil
}
