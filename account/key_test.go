package account

import (
	"go-vite/crypto/ed25519"
	"testing"
)

func TestStoreNewKey(t *testing.T) {

	ks := keyStorePassphrase{keysDirPath: "/Users/zhutiantao"}
	key, addr, err := StoreNewKey(ks, "123456")
	if err != nil {
		t.Fatal(err)
	}
	println(key.PrivateKey.HexStr())
	println(addr.Hex())

	key, err = ks.ExtractKey(addr, "123456")
	if err != nil {
		t.Fatal(err)
	}

	println(key.PrivateKey.HexStr())
	println(addr.Hex())

}

func TestEncryptAndDecrypt(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	key := newKeyFromEd25519(&priv)

	json, err := EncryptKey(key, "123456")
	if err != nil {
		t.Fatal(err)
	}

	key1, err := DecryptKey(json, "123456")
	if err != nil {
		t.Fatal(err)
	}

	if key.PrivateKey.HexStr() != key1.PrivateKey.HexStr() {
		t.Fatalf("key not equal")
	}

}
