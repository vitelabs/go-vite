package crypto

import (
	"encoding/hex"
	"testing"

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
	keyArray := []byte(gcm_dummy_key_32)
	plain := []byte(gcm_dummy_plain_text)
	out, nonce, err := AesGCMEncrypt(keyArray, plain)
	println(hex.EncodeToString(out))
	println(hex.EncodeToString(nonce))
	println("Encrypt finish")
	if err != nil {
		t.Fatal(err)
	}

	plain1, err := AesGCMDecrypt(keyArray, out, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain1, plain) {
		t.Fatal("Mis content")
	}

}

func TestAesCTRXOR(t *testing.T) {
	keyArray := []byte(gcm_dummy_key_32)
	plainArray := []byte(gcm_dummy_plain_text)
	iv := []byte(gcm_dummy_key_16)
	cipher, err := AesCTRXOR(keyArray, plainArray, iv)
	if err != nil {
		t.Fatal(err)
	}
	println("Encrypt finish")
	plainArray1, err := AesCTRXOR(keyArray, cipher, iv)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(plainArray, plainArray1) {
		t.Fatal("mis content")
	}

	println("plainArray :", string(plainArray))
	println("plainArray1:", string(plainArray1))
	println(hex.EncodeToString(cipher))

}
