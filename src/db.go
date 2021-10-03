package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	"os"
)

func GetDB() (*dynamo.DB, error) {
	sesh := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return dynamo.New(sesh, &aws.Config{Region: aws.String(os.Getenv("AWS_REGION"))}), nil
}
