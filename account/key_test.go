package account

import (
	"testing"
	"go-vite/common"
)

var address common.Address

func TestStoreNewKey(t *testing.T) {

	ks := keyStorePassphrase{keysDirPath: "/Users/zhutiantao"}
	key, addr, err := StoreNewKey(ks, "123456")
	if err != nil {
		t.Fatal(err)
	}
	println(key.PrivateKey.HexStr())
	println(addr.Hex())
	address = addr
	//key, err = ks.ExtractKey(addr, "123456")
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//println(key.PrivateKey.HexStr())
	//println(addr.Hex())

}
func TestExtractNewKey(t *testing.T) {

	ks := keyStorePassphrase{keysDirPath: "/Users/zhutiantao"}

	key, err := ks.ExtractKey(address, "123456")
	if err != nil {
		t.Fatal(err)
	}

	println(key.PrivateKey.HexStr())
	println(address.Hex())

}