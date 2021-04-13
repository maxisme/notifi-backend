package crypt

import (
	"testing"
)

var testKey = []byte("lKmxnmQ[ATrrj4eE$WHUnBotIwSy8boe")
var testB64PublicKey = "LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tDQpNSUlCQ2dLQ0FRRUExbzRmblJyd1R0UkpuOUhaS3BhWFdrT3RLNFV5WUdlSTg5dWk5MjhyekxaS3hTRXVvc2lVUm1vczIwRFU3SmtvYW5UTllsWUdud3AwakpMZVdOQVZYUGUyYjhia292dmNSVTMvUFQzZjJjVXRsb28zUHZqZGRPeXZnajBQWGFFUG9IOWtiRTNJaXhkcVJQaVRvN3FzaG5FYzFhdEErY3p0TEVXRk00R21VM3BRbkFCSHRRbDRVUHhCZEM1VHl1U1Blbys4Y2szUDg1aEtxcmRKRnBtbC9vM1RQMm5NcTJBaW1nQTdsWE8zSlh0ZXdZaFdoTHRoN2lmK1h2U1EyM2JPeTc2dzlFNGdqcUlRdHU5dmNBeHVzQU5PMlZlbGRXYU1BYzRSSDFESDk4OFowOWFOTXBWUSt2eFNoZlZhRkZTdldBV0tmOTVybTMrSC9UL0NUeklqVXdJREFRQUINCi0tLS0tRU5EIFJTQSBQVUJMSUMgS0VZLS0tLS0="

// var testB64PrivateKey = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQ0KTUlJRnBBSUJBQUtDQVFFQTFvNGZuUnJ3VHRSSm45SFpLcGFYV2tPdEs0VXlZR2VJODl1aTkyOHJ6TFpLeFNFdW9zaVVSbW9zMjBEVTdKa29hblROWWxZR253cDBqSkxlV05BVlhQZTJiOGJrb3Z2Y1JVMy9QVDNmMmNVdGxvbzNQdmpkZE95dmdqMFBYYUVQb0g5a2JFM0lpeGRxUlBpVG83cXNobkVjMWF0QStjenRMRVdGTTRHbVUzcFFuQUJIdFFsNFVQeEJkQzVUeXVTUGVvKzhjazNQODVoS3FyZEpGcG1sL28zVFAybk1xMkFpbWdBN2xYTzNKWHRld1loV2hMdGg3aWYrWHZTUTIzYk95NzZ3OUU0Z2pxSVF0dTl2Y0F4dXNBTk8yVmVsZFdhTUFjNFJIMURIOTg4WjA5YU5NcFZRK3Z4U2hmVmFGRlN2V0FXS2Y5NXJtMytIL1QvQ1R6SWpVd0tDQVFFQXdlNGlxeG1mWGxGSjN2aUp4NUhvYWtGRHRGT25yalhITjB3dWFHS0cvM2xCNmg3TkRYd3BjZUFrZldXRFQvdzc2TVY4bzBiSW8xYUs5RWtJU3RTQ2lzNm9peTRHbVQ3MzRYalhuUjVoU0hDT2ZVU2ZIUDlMQkpXdktoUlE0RHRsYVpmY1NIeWlLUDdZSWxGZytad1F0MUJrVk1sL2FJc1BVWFhoS1NZbUZjcVFRWDhNMURKejBEU1M3L0ptYklEYjJFQnNTU3p1YWtkd1lEMHNxaDVyU2kvVlFhVTVuaW1oZkk2MFByVXg4QU5wckhHUnYrUzFPdDNmbUZPZDhiMHVJMXFIYkNJR2lqaStqVExndlQrYWI2Y3NtTjdzUE45NVNhQXRsNmtjdXZaVkFuZ2tLaDdoVWNta0N0c3dTUUt3LzBFUDhUTmxFL1pyazB4UWs0VlArUUtDQVFFQXdlNGlxeG1mWGxGSjN2aUp4NUhvYWtGRHRGT25yalhITjB3dWFHS0cvM2xCNmg3TkRYd3BjZUFrZldXRFQvdzc2TVY4bzBiSW8xYUs5RWtJU3RTQ2lzNm9peTRHbVQ3MzRYalhuUjVoU0hDT2ZVU2ZIUDlMQkpXdktoUlE0RHRsYVpmY1NIeWlLUDdZSWxGZytad1F0MUJrVk1sL2FJc1BVWFhoS1NZbUZjcVFRWDhNMURKejBEU1M3L0ptYklEYjJFQnNTU3p1YWtkd1lEMHNxaDVyU2kvVlFhVTVuaW1oZkk2MFByVXg4QU5wckhHUnYrUzFPdDNmbUZPZDhiMHVJMXFIYkNJR2lqaStqVExndlQrYWI2Y3NtTjdzUE45NVNhQXRsNmtjdXZaVkFuZ2tLaDdoVWNta0N0c3dTUUt3LzBFUDhUTmxFL1pyazB4UWs0VlArUUtCZ1FEeTYrOFdJUHFJRVN1a2pFcExvdWhxanJyWmFYU3EwSzZSKy9nTXVURTNSUVl5VTVUb2VQUFMxTkJtMU0rQnl6MEFJQnR4RnQ3WE12YzNnNzkxM2Z4S0NGcklQendIRWpQbFBqZDlQb1c5VTJiRFl0U3ROdHZ2cmV6eGlLR2pwVUJOa09CT1FPZzZ0NTI0dVpRa2dkaVUxWXZxclFVVkdXcDcvc0NvMFlMWTF3S0JnUURpR3pxb2dpUENrQU9QVm9iMG1ORWxtb0tzbmxDZ2N0djhCZ1l1RTRtL0EzZkp0Z0R5L3pHSTQycHYvTjJkTTBHNm1ZTHRVR2wzRjEyeG1PdmxYYXhONG96MDdpbG5OWm9mSjdtaTF0cEpqMEVFR3V6dVhhOHZFV1M5NTBEaDgxK3dJTy9tSkdjMFZJTjZXUWFsREREQWJ5N2diU3NhRVpWNWVyYWYxRTdONVFLQmdRQzhsQ09hamlkdkVjVkxqQXp4QXBwaWZrTFhIR0tSYUViYzFUb094b1ZLWHE4Y3luM0Nxb0s5bksvYjVFRGloWi9wUlFPSW16U0s0dW0va3V0QzJQMU5pNGJPQlNqWVpHMGMvVnVlUXJjWDduTE1JeUR2QnJOZU1Tckxwa0cxQkVnUzd2RHlUcmo1UENtWWlaaFRidWx2UVFmSk9sL0RyV05ZdHI5aFRxUEJLUUtCZ0ZvNER6SEpxOGNvZTZNb0hYVmZ2S1JLZ0xXci9mUG5vTXR4QStwQ3RZWFlObVh3RDNUbVNyZWFOcEEwejZDNElSUDV5UG8wU2NEUk8vdHZUMEVZSFhaK1hVd2w4N05RK2d4UVo0d0lPdFY3S2JBZnBrWitielpTdEdYcDdrTzZQb1lpdmxhUVUvWFhleGJJaXhRMFJ3ZWgxWXlMUXRXR0NxU01TRzZCNG1mWkFvR0FaU0paZjBlMEdTeUJxbE1admpQZ1Q2SFlUaUUzelh4SlhSTFZEWnp1eHVjVEpBSGZvcStOVUw0VCtWUHB1VmVaY2hsWmdScG1CQ1NGWTBVM01VUUxHTU0xTTNNTDBCYWJKYXBqTXI3ZktuM2RUV1d1VXl3cG91M1hVTjdkRGxKR3h4aTR2Y2E0MzVBZElScUQ2MythMW9TeTQ1MHZCemVqYWZlL1VVT0xyamM9DQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQ==";

var testStr = RandomString(10)

func TestEncrypt(t *testing.T) {
	encryptedstr, err := EncryptAES(testStr, testKey)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	decryptedstr, err := DecryptAES(encryptedstr, testKey)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if decryptedstr != testStr {
		t.Errorf("%s does not equal %s", decryptedstr, testStr)
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

func TestB64StringToPubKey(t *testing.T) {
	_, err := B64StringToPubKey(testB64PublicKey)
	if err != nil {
		t.Errorf("key should be valid: " + err.Error())
	}
}

func TestEncryptWithPubKey(t *testing.T) {
	key, err := B64StringToPubKey(testB64PublicKey)
	if err != nil {
		t.Errorf("key should be valid: " + err.Error())
	}
	encryptedStr, err := EncryptWithPubKey("foo", key)
	if err != nil {
		t.Errorf("key should be valid: " + err.Error())
	}
	print(encryptedStr)
}
