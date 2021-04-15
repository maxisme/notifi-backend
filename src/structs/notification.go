package structs

import "github.com/maxisme/notifi-backend/crypt"

// Notification structure
type Notification struct {
	Credentials  string `json:"credentials"`
	UUID         string `json:"UUID"`
	Time         string `json:"time"`
	Title        string `json:"title"`
	Message      string `json:"message"`
	Image        string `json:"image"`
	Link         string `json:"link"`
	EncryptedKey string `json:"encrypted_key"`
}

// Encrypt
func (n Notification) Encrypt(b64PubKey string) error {
	key := []byte(crypt.RandomString(32))

	var err error
	n.Title, err = crypt.EncryptAES(n.Title, key)
	if err != nil {
		return err
	}

	n.Message, err = crypt.EncryptAES(n.Message, key)
	if err != nil {
		return err
	}

	n.Image, err = crypt.EncryptAES(n.Image, key)
	if err != nil {
		return err
	}

	n.Link, err = crypt.EncryptAES(n.Link, key)
	if err != nil {
		return err
	}

	// encrypt key with rsa pub key
	RSAKey, err := crypt.B64StringToPubKey(b64PubKey)
	if err != nil {
		return err
	}
	n.EncryptedKey, err = crypt.EncryptWithPubKey(key, RSAKey)
	return err
}
