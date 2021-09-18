package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"os"
)

func main() {
	switch arg := os.Args[1]; arg {
	case "code":
		lambda.Start(HandleCode)
	case "connect":
		lambda.Start(HandleConnect)
	case "message":
		lambda.Start(HandleMessage)
	case "disconnect":
		lambda.Start(HandleDisconnect)
	case "api":
		lambda.Start(HandleApi)
	default:
		panic("missing args")
	}
}
