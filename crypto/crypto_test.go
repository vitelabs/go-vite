package crypto

import (
	"encoding/hex"
	"testing"

	"bytes"
	"fmt"

	"github.com/vitelabs/go-vite/crypto/ed25519"
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
	println("key:", hex.EncodeToString(keyArray))
	println("origin:", hex.EncodeToString(plain))
	println("cipher:", hex.EncodeToString(out))
	println("nonce:", hex.EncodeToString(nonce))
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
		t.Error("mis content")
	}

	println("plainArray :", hex.EncodeToString(plainArray))
	println("plainArray1:", hex.EncodeToString(plainArray1))
	println(hex.EncodeToString(cipher))

}

func TestGenerateKey(t *testing.T) {
	for i := 0; i < 5; i++ {
		publicKey, privateKey, _ := ed25519.GenerateKey(nil)
		pub := hex.EncodeToString(publicKey)
		println(pub)
		pri := hex.EncodeToString(privateKey)
		println(pri)
	}

	key, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	publicKey, privateKey, _ := ed25519.GenerateKey(bytes.NewReader(key))
	fmt.Println(hex.EncodeToString(publicKey))
	fmt.Println(hex.EncodeToString(privateKey))

}

func TestX25519ComputeSecret(t *testing.T) {
	pub1, priv1, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Errorf("Failed to generate ed25519 key: %v", err)
	}
	fmt.Println("ed25519", hex.EncodeToString(pub1), hex.EncodeToString(priv1))

	cpub1, cpriv1 := pub1.ToX25519Pk(), priv1.ToX25519Sk()
	fmt.Println("curve25519", hex.EncodeToString(cpub1), hex.EncodeToString(cpriv1))

	pub2, priv2, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Errorf("Failed to generate ed25519 key: %v", err)
	}
	fmt.Println("ed25519", hex.EncodeToString(pub2), hex.EncodeToString(priv2))

	cpub2, cpriv2 := pub2.ToX25519Pk(), priv2.ToX25519Sk()
	fmt.Println("curve25519", hex.EncodeToString(cpub2), hex.EncodeToString(cpriv2))

	b1, err := X25519ComputeSecret(cpriv1, cpub2)
	if err != nil {
		t.Errorf("Failed to computed x25519 secret: %v", err)
	}

	b2, err := X25519ComputeSecret(cpriv2, cpub1)
	if err != nil {
		t.Errorf("Failed to computed x25519 secret: %v", err)
	}

	if !bytes.Equal(b1, b2) {
		t.Error("Different secret")
	}

	fmt.Println("x25519 secret", hex.EncodeToString(b1))
}
