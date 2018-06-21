package crypto

import (
	"testing"
	"encoding/hex"

	"bytes"
)

const (
	long_bytes           = 100000
	gcm_dummy_plain_text = "112233445566778899AAABBCCBC"
	gcm_dummy_key_16     = "1122334455667788"
	gcm_dummy_key_32     = "11112222333344445555666677778888"
	gcm_dummy_key_24     = "111222333444555666777888"
)

func TestHash256(t *testing.T) {

	println(hex.EncodeToString(Hash256([]byte{1, 2, 3})))
	println(hex.EncodeToString(Hash256([]byte{})))
	println(hex.EncodeToString(Hash256(nil)))
	LongBytes := make([]byte, long_bytes)
	for i := 0; i < long_bytes; i++ {
		LongBytes[i] = 21
	}
	println(hex.EncodeToString(Hash256(LongBytes)))

}

func TestAesGCMEncrypt(t *testing.T) {
	keyArray := []byte(gcm_dummy_key_16)
	plainArray := []byte(gcm_dummy_plain_text)
	out, nonce, err := AesGCMEncrypt(keyArray, plainArray)
	println(hex.EncodeToString(out))
	println(hex.EncodeToString(nonce))
	if err != nil {
		t.Fatalf("AES GCM Encrypt ERROR %s", err.Error())
	}

	plain, err := AesGCMDecrypt(keyArray, out, nonce)
	if err != nil {
		t.Fatalf("AES GCM Decrypt ERROR %s", err.Error())
	}
	if !bytes.Equal(plain, plainArray) {
		t.Fatal("AES GCM Decrypt ERROR NOT EQUAL")
	}

}

func TestGoSyntax(t *testing.T) {
	b := []byte{'1', '2', '3'}

	println(string(b[:2]))
	println(string(b[1:3]))
}
