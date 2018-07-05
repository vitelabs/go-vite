package keystore

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"testing"
)

const (
	TEST_KEY_JSON                = `{"hexaddress":"vite_f8e492c13b4a3fad540ebdb60deddf14be8bc0db95eabb78bc","crypto":{"ciphername":"aes-256-gcm","ciphertext":"28afbdb3ecd485b1f3aa6c58ff4e216a317e842629298e3496301a846f9cb117b275f29f8465369ac49b79dc99699b6d2dacea31d44b0c37f65549e505a50050984fece97a4f8215b9ba6b1e4ba7c2e4","nonce":"1686f80ef133c94b77fca937","kdf":"scrypt","scryptparams":{"n":262144,"r":8,"p":1,"keylen":32,"salt":"21f156ab3268578ea35c58f94a8687505eb4f4895061b2bea41acf9388d97a01"}},"id":"9c079e8a-bd7b-4de3-bd4d-cc8ab1233e3b","keystoreversion":1}`
	TEST_KEY_JSON_ADDR_NOT_EQUAL = `{"hexaddress":"vite_bcdc5b9dd0ed0de7de2f0e97c36638e108aa64a2bedc22c0e6","crypto":{"ciphername":"aes-256-gcm","ciphertext":"28afbdb3ecd485b1f3aa6c58ff4e216a317e842629298e3496301a846f9cb117b275f29f8465369ac49b79dc99699b6d2dacea31d44b0c37f65549e505a50050984fece97a4f8215b9ba6b1e4ba7c2e4","nonce":"1686f80ef133c94b77fca937","kdf":"scrypt","scryptparams":{"n":262144,"r":8,"p":1,"keylen":32,"salt":"21f156ab3268578ea35c58f94a8687505eb4f4895061b2bea41acf9388d97a01"}},"id":"9c079e8a-bd7b-4de3-bd4d-cc8ab1233e3b","keystoreversion":1}`
	TEST_PRIKEY                  = "eaa252795874fc5c7c3b1321954258ae4f469a498d60362c298dc51532c06b78b1f4b94b47b524e25946040ad63227fa28c64c05d5ff97058a7894119e1385f3"
)

func TestDecrypt(t *testing.T) {
	{
		key1, err := DecryptKey([]byte(TEST_KEY_JSON), "123456")
		if err != nil {
			t.Fatal(err)
		}

		if TEST_PRIKEY != key1.PrivateKey.Hex() {
			t.Fatalf("key not equal")
		}
	}

	{
		_, err := DecryptKey([]byte(TEST_KEY_JSON_ADDR_NOT_EQUAL), "123456")
		if err == nil {
			t.Fatal("")
		}
	}
}

func TestEncryptAndDecrypt(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	key := newKeyFromEd25519(&pub, &priv)

	json, err := EncryptKey(key, "123456")
	if err != nil {
		t.Fatal(err)
	}

	key1, err := DecryptKey(json, "123456")
	if err != nil {
		t.Fatal(err)
	}

	println("K0 generate Address:" + key.Address.Hex())
	println("K1 generate Address:" + types.PrikeyToAddress(*key1.PrivateKey).Hex())

	println("K0 generate Prikey:" + key.PrivateKey.Hex())
	println("K1 generate Prikey:" + key1.PrivateKey.Hex())

	if key.PrivateKey.Hex() != key1.PrivateKey.Hex() {
		t.Fatalf("key not equal")
	}

	println(string(json))

}
