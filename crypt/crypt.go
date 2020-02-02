package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	r "crypto/rand"
	"crypto/sha256"
	b64 "encoding/base64"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log"
	"math/rand"
)

func EncryptAES(str string, key []byte) (string, error) {
	if len(str) == 0 {
		return "", nil
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(r.Reader, nonce); err != nil {
		return "", err
	}

	return b64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(str), nil)), nil
}

func DecryptAES(encryptedstr string, key []byte) (string, error) {
	if len(encryptedstr) == 0 {
		return "", nil
	}

	encryptedbytes, _ := b64.StdEncoding.DecodeString(encryptedstr)
	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedbytes) < nonceSize {
		return "", err
	}

	nonce, ciphertext := encryptedbytes[:nonceSize], encryptedbytes[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func RandomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func Hash(str string) string {
	return HashWithBytes([]byte(str))
}

func HashWithBytes(str []byte) string {
	v := sha256.Sum256(str)
	return b64.StdEncoding.EncodeToString(v[:])
}

func PassHash(str string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.MinCost)
	if err != nil {
		log.Println(err.Error())
	}
	return string(hash)
}

func VerifyPassHash(str string, expectedstr string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(str), []byte(expectedstr))
	return err == nil
}
