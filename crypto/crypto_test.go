package crypto

import (
	"testing"
	"encoding/hex"
	"bytes"
)

const (
	LONG_BYTES = 100000
)

func TestHash256(t *testing.T) {

	println(hex.EncodeToString(Hash256([]byte{1, 2, 3})))
	println(hex.EncodeToString(Hash256([]byte{})))
	println(hex.EncodeToString(Hash256(nil)))
	LongBytes := make([]byte, LONG_BYTES)
	for i := 0; i < LONG_BYTES; i++ {
		LongBytes[i] = 21
	}
	println(hex.EncodeToString(Hash256(LongBytes)))

}

func TestCreateRandomAddress(t *testing.T) {
	addr_, priv, _ := CreateRandomAddress()
	println(addr_.Str())

	pubkey := recoverPubKey(priv)

	addr2 := PubkeyToAddress(pubkey[:])
	println(addr2.Str())

	if !bytes.Equal(addr_[:], addr2[:]) {
		t.Fail()
	}
}

func TestAddressValid(t *testing.T) {
	{
		fakeAddr := "1231231"
		if isValidAddress([]byte(fakeAddr)) {
			t.Fail()
		}
	}
	{
		if isValidAddress(nil) {
			t.Fail()
		}
	}

	{
		fakeAddr := "vite_bcdc5b9dd0ed0de7de2f0e97c36638e108aa64a2bedc22c0e7"
		if isValidAddress([]byte(fakeAddr)) {
			t.Fail()
		}

	}
	{
		fakeAddr := "vite_asdc5b9dd0ed0de7de2f0e97c36638e108aa64a2bedc22c0e6"
		if isValidAddress([]byte(fakeAddr)) {
			t.Fail()
		}

	}

	{
		fakeAddr := "aite_asdc5b9dd0ed0de7de2f0e97c36638e108aa64a2bedc22c0e6"
		if isValidAddress([]byte(fakeAddr)) {
			t.Fail()
		}

	}


	{
		realAddr := "vite_bcdc5b9dd0ed0de7de2f0e97c36638e108aa64a2bedc22c0e6"
		if !isValidAddress([]byte(realAddr)) {
			t.Fail()
		}

	}
}

func TestGoSyntax(t *testing.T) {
	b := []byte{'1', '2', '3'}

	println(string(b[:2]))
	println(string(b[1:3]))
}
