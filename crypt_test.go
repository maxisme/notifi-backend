package main

import (
	"os"
	"testing"
)

var testKey = []byte(os.Getenv("encryption_key"))
var testStr = RandomString(10)

func TestEncrypt(t *testing.T) {
	encryptedstr := Encrypt(testStr, testKey)
	decryptedstr, _ := Decrypt(encryptedstr, testKey)
	if decryptedstr != testStr {
		t.Errorf("Encryption did not work!")
	}
}

func TestInvalidtest_key(t *testing.T) {
	test_key2 := []byte(RandomString(10))
	encryptedstr := Encrypt(testStr, testKey)
	_, err := Decrypt(encryptedstr, test_key2)
	if err == nil {
		t.Errorf("Invalid test_key did not break!")
	}
}

func TestInvalidString(t *testing.T) {
	testenryptedstr := RandomString(10)
	str, _ := Decrypt(testenryptedstr, testKey)
	if str != "" {
		t.Errorf("Invalid string did not break!")
	}
}

func TestHash(t *testing.T) {
	if len(Hash(RandomString(10))) != 44 {
		t.Errorf("Hash algo not working as expected")
	}
	if Hash(RandomString(10)) == Hash(RandomString(10)) {
		t.Errorf("Hash is not hashing properly")
	}
	str := RandomString(10)
	if Hash(str) != Hash(str) {
		t.Errorf("Hash is not hashing properly 2")
	}
}
