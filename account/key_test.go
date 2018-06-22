package account

import (
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
