package crypt

import (
	"testing"
)

var testKey = []byte("lKmxnmQ[ATrrj4eE$WHUnBotIwSy8be6oe")
var testStr = RandomString(10)

func TestEncrypt(t *testing.T) {
	encryptedstr, _ := EncryptAES(testStr, testKey)
	decryptedstr, _ := DecryptAES(encryptedstr, testKey)
	if decryptedstr != testStr {
		t.Errorf("Encryption did not work! You probably did not set env variable - ENCRYPTION_KEY = steal from github actions ;) ")
	}
}

func TestInvalidtest_key(t *testing.T) {
	testKey2 := []byte(RandomString(10))
	encryptedstr, _ := EncryptAES(testStr, testKey)
	_, err := DecryptAES(encryptedstr, testKey2)
	if err == nil {
		t.Errorf("Invalid test_key did not break!")
	}
}

func TestInvalidString(t *testing.T) {
	testenryptedstr := RandomString(10)
	str, _ := DecryptAES(testenryptedstr, testKey)
	if str != "" {
		t.Errorf("Invalid string did not break!")
	}
}

func TestHash(t *testing.T) {
	if len(Hash(RandomString(10))) != 44 {
		t.Errorf("Hash algo not working as expected")
	}
}

func TestPassHash(t *testing.T) {
	passwordStr := RandomString(10)
	passwordHash := PassHash(passwordStr)
	passwordHash2 := PassHash(passwordStr)

	if passwordHash == passwordHash2 {
		t.Errorf("hashed passwords should be different")
	}

	if VerifyPassHash(passwordHash, passwordHash2) {
		t.Errorf("password should have verified successfully")
	}
}
