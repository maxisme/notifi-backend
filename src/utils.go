package main

import (
	_ "database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/guregu/dynamo"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const (
	Region      = "us-east-1"
	WSStageName = "ws"
)

//func NewDynamoDBSession() *dynamodb.DynamoDB {
//	sess, _ := session.NewSession(&aws.Config{
//		Region:      aws.String(Region),
//		Credentials: awscreds.NewStaticCredentials(AccessKeyID, SecretAccessKey, ""),
//	})
//	return dynamodb.New(sess)
//}

func NewAPIGatewaySession(endpoint string) *apigatewaymanagementapi.ApiGatewayManagementApi {
	sesh := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region:   aws.String(Region),
			Endpoint: aws.String(endpoint),
		},
	}))
	return apigatewaymanagementapi.New(sesh)
}

func WriteError(err error, code int) (events.APIGatewayProxyResponse, error) {
	_, file, no, _ := runtime.Caller(1)
	fmt.Printf("%s#%d: request error: %s %d\n", file, no, err.Error(), code)
	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Body:       err.Error(),
	}, err
}

func WriteEmptySuccess() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil
}

//const rfc2822 = "Mon, 28 Jan 2013 14:30:00 +0500"
//
//type GitHubResponse struct {
//	TagName     string    `json:"tag_name"`
//	Name        string    `json:"name"`
//	Prerelease  bool      `json:"prerelease"`
//	Draft       bool      `json:"draft"`
//	CreatedAt   time.Time `json:"created_at"`
//	PublishedAt time.Time `json:"published_at"`
//	Assets      []struct {
//		Name               string `json:"name"`
//		Size               int    `json:"size"`
//		BrowserDownloadURL string `json:"browser_download_url"`
//	} `json:"assets"`
//	Body string `json:"body"`
//}

//// RequiredEnvs verifies envKeys all have values
//func RequiredEnvs(envKeys []string) error {
//	for _, envKey := range envKeys {
//		envValue := os.Getenv(envKey)
//		if envValue == "" {
//			return fmt.Errorf("missing env variable: '%s'", envKey)
//		}
//	}
//	return nil
//}
//
//// GetGitHubResponses parses json from http response
//func GetGitHubResponses(url string) ([]GitHubResponse, error) {
//	const cacheKey = "github-response"
//
//	if githubResponses, found := c.Get(cacheKey); found {
//		return githubResponses.([]GitHubResponse), nil
//	}
//
//	var client = &http.Client{Timeout: 2 * time.Second}
//	r, err := client.Get(url)
//	if err != nil {
//		return nil, err
//	}
//
//	defer r.Body.Close()
//
//	var githubResponses []GitHubResponse
//	err = json.NewDecoder(r.Body).Decode(&githubResponses)
//	if err != nil {
//		return nil, err
//	}
//
//	err = c.Add(cacheKey, githubResponses, cache.DefaultExpiration)
//	if err != nil {
//		return nil, err
//	}
//	return githubResponses, err
//}
//
//func GetWSChannelKey(channel string) string {
//	return crypt.Hash(channel)
//}

func SendWsMessage(connectionID string, msgData []byte) error {
	connectionInput := &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionID),
		Data:         msgData,
	}

	endpoint := strings.Replace(os.Getenv("WS_ENDPOINT"), "wss://", "https://", 1) + "/" + WSStageName
	fmt.Println(endpoint)
	//endpoint := "https://" + requestContext.DomainName + "/" + requestContext.Stage
	_, err := NewAPIGatewaySession(endpoint).PostToConnection(connectionInput)
	return err
}

func SendStoredMessages(db *dynamo.DB, credentials string, requestContext events.APIGatewayWebsocketProxyRequestContext) error {
	result, err := GetItems(db, NotificationTable, "credentials", credentials)
	if err != nil {
		return err
	}

	notifications, ok := result.([]Notification)
	if ok && len(notifications) > 0 {
		bytes, err := json.Marshal(notifications)
		if err != nil {
			return err
		}
		if err := SendWsMessage(requestContext.ConnectionID, bytes); err != nil {
			return err
		}
	}
	return nil
}
