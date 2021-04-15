package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/maxisme/notifi-backend/structs"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
)

const notifiURL = "https://dev.notifi.it/"

func main() {
	var n Notification
	var sendCmd = &cobra.Command{
		Use:   "send",
		Short: "Send an encrypted notification",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Encrypting...")
			key, err := _getPubKey(n.Credentials)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if err := n.Encrypt(key); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Println("Sending...")
			_, err = _sendMsg(n)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Println("Sent")
		},
	}

	sendCmd.Flags().StringVarP(&n.Credentials, "credentials", "c", "", "Credentials of user to receive notification")
	sendCmd.Flags().StringVarP(&n.Title, "title", "t", "", "Title for notification")
	sendCmd.Flags().StringVarP(&n.Message, "message", "m", "", "Message for notification")
	sendCmd.Flags().StringVarP(&n.Image, "image", "i", "", "Image URL for notification")
	sendCmd.Flags().StringVarP(&n.Link, "link", "l", "", "URL link for notification")
	_ = sendCmd.MarkFlagRequired("credentials")
	_ = sendCmd.MarkFlagRequired("title")

	if err := sendCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func _sendMsg(notification Notification) (*http.Response, error) {
	b, err := json.Marshal(notification)
	if err != nil {
		return nil, err
	}
	return http.Post(notifiURL+"api", "application/json", bytes.NewReader(b))
}

func _getPubKey(c string) (string, error) {
	resp, err := http.Get(notifiURL + "key?credentials=" + c)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(bodyBytes), nil
	}
	return "", errors.New("Unable to get key for credentials")
}
