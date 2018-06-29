package keystore

import (
	"testing"
)

func TestStoreAndExtractNewKey(t *testing.T) {

	ks := keyStorePassphrase{keysDirPath: DefaultDataDir()}
	kp := NewKeyPoll(&DefaultKeyConfig)

	key1, addr1, err := kp.StoreNewKey("123456")
	if err != nil {
		t.Fatal(err)
	}

	println("Encrypt finish")

	key2, err := ks.ExtractKey(addr1, "123456")
	if err != nil {
		t.Fatal(err)
	}
	println("decrypt finish")
	println(key1.PrivateKey.HexStr())
	println(key2.PrivateKey.HexStr())

	if key1.PrivateKey.HexStr() != key2.PrivateKey.HexStr() {
		t.Fatalf("miss PrivateKey content")
	}

}

