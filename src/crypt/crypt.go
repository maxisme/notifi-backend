package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	b64 "encoding/base64"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log"
	"math/big"
)

// EncryptAES encrypts a string using AES with a key
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
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	return b64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(str), nil)), nil
}

// DecryptAES decrypts a string using AES with a key
func DecryptAES(str string, key []byte) (string, error) {
	if len(str) == 0 {
		return "", nil
	}

	encryptedbytes, _ := b64.StdEncoding.DecodeString(str)
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

// RandomString generates a random string
func RandomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		randNumber, err := rand.Int(rand.Reader, big.NewInt(int64(len(letter))))
		if err != nil {
			panic(err)
		}
		b[i] = letter[randNumber.Int64()]
	}
	return string(b)
}

// Hash hashes a string TODO cache
func Hash(str string) string {
	return HashWithBytes([]byte(str))
}

// HashWithBytes hashes bytes
func HashWithBytes(str []byte) string {
	v := sha256.Sum256(str)
	return b64.StdEncoding.EncodeToString(v[:])
}

// PassHash hashes a string specifically to be used for passwords
func PassHash(str string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.MinCost)
	if err != nil {
		log.Println(err.Error())
	}
	return string(hash)
}

// VerifyPassHash verifies that two hashed strings are for the same password string
func VerifyPassHash(str string, expectedStr string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(str), []byte(expectedStr))
	return err == nil
}
