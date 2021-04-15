package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/pem"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log"
	"math/big"
)

// EncryptAES encrypts a string using AES with a key
func EncryptAES(msg string, key []byte) (string, error) {
	if len(msg) == 0 {
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

	return b64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(msg), nil)), nil
}

// DecryptAES decrypts a string using AES with a key
func DecryptAES(msg string, key []byte) (string, error) {
	if len(msg) == 0 {
		return "", nil
	}

	encryptedbytes, _ := b64.StdEncoding.DecodeString(msg)
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

// B64StringToPubKey converts a base64 RSA string to a public key
// TODO cache
func B64StringToPubKey(b64PubKey string) (*rsa.PublicKey, error) {
	key, err := b64.StdEncoding.DecodeString(b64PubKey)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(key)
	re, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return re, nil
}

// EncryptWithPubKey encrypts a string with public rsa key using a random key
func EncryptWithPubKey(msg []byte, pub *rsa.PublicKey) (string, error) {
	encryptedKey, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, pub, msg, nil)
	if err != nil {
		return "", err
	}
	return b64.StdEncoding.EncodeToString(encryptedKey), err
}
