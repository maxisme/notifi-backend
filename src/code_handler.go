package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
)

func HandleCode(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "hi5",
	}, nil
	//if r.HTTPMethod != "POST" {
	//	return WriteError(fmt.Errorf("Method not allowed"), http.StatusBadRequest)
	//}
	//
	//// create PostUser struct
	//PostUser := User{
	//	UUID: r.StageVariables["UUID"],
	//
	//	// if asking for new Credentials
	//	Credentials: Credentials{
	//		Value: r.StageVariables["current_credentials"],
	//		Key:   r.StageVariables["current_credential_key"],
	//	},
	//}
	//
	//firebaseToken, ok := r.StageVariables["firebase_token"]
	//if ok {
	//	PostUser.FirebaseToken = firebaseToken
	//}
	//
	//if !IsValidUUID(PostUser.UUID) {
	//	return WriteError(fmt.Errorf("Invalid UUID"), http.StatusBadRequest)
	//}
	//
	//db, err := GetDB()
	//if err != nil {
	//	return WriteError(err, http.StatusInternalServerError)
	//}
	//
	//creds, err := PostUser.Store(db)
	//if err != nil {
	//	return WriteError(err, http.StatusInternalServerError)
	//}
	//
	//c, err := json.Marshal(creds)
	//if err != nil {
	//	return WriteError(err, http.StatusInternalServerError)
	//}
	//
	//return events.APIGatewayProxyResponse{
	//	StatusCode: http.StatusOK,
	//	Body:       string(c),
	//}, nil
}
