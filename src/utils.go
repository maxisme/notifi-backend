package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/maxisme/notifi-backend/crypt"
	"github.com/patrickmn/go-cache"
	"net/http"
	"os"
	"time"
)

var c = cache.New(5*time.Minute, 10*time.Minute)

const rfc2822 = "Mon, 28 Jan 2013 14:30:00 +0500"

type GitHubResponse struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		Size               int    `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	Body string `json:"body"`
}

// RequiredEnvs verifies envKeys all have values
func RequiredEnvs(envKeys []string) error {
	for _, envKey := range envKeys {
		envValue := os.Getenv(envKey)
		if envValue == "" {
			return fmt.Errorf("missing env variable: '%s'", envKey)
		}
	}
	return nil
}

// UpdateErr returns an error if no rows have been effected
func UpdateErr(res sql.Result, err error) error {
	if err != nil {
		return err
	}

	rowsEffected, err := res.RowsAffected()
	if rowsEffected == 0 {
		return errors.New("no rows effected")
	}
	return err
}

// GetGitHubResponses parses json from http response
func GetGitHubResponses(url string) ([]GitHubResponse, error) {
	const cacheKey = "github-response"

	if githubResponses, found := c.Get(cacheKey); found {
		return githubResponses.([]GitHubResponse), nil
	}

	var client = &http.Client{Timeout: 2 * time.Second}
	r, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	var githubResponses []GitHubResponse
	err = json.NewDecoder(r.Body).Decode(&githubResponses)
	if err != nil {
		return nil, err
	}

	err = c.Add(cacheKey, githubResponses, cache.DefaultExpiration)
	if err != nil {
		return nil, err
	}
	return githubResponses, err
}

func GetWSChannelKey(channel string) string {
	return crypt.Hash(channel)
}
