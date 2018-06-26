package common

import (
	"bytes"
	"testing"
)

func TestCreateRandomAddress(t *testing.T) {
	addr, priv, _ := CreateAddress()
	println(addr.String())

	pubkey := priv.PubByte()

	addr2 := PubkeyToAddress(pubkey[:])
	println(addr2.String())

	if !bytes.Equal(addr[:], addr2[:]) {
		t.Fail()
	}
}

func TestCreateDAddress(t *testing.T) {
	var zero [32]byte
	addr_, priv, _ := CreateAddressWithDeterministic(zero)
	addr_1, priv1, _ := CreateAddressWithDeterministic(zero)

	if !bytes.Equal(addr_[:], addr_1[:]) {
		t.Fatalf("addr create error")
	}

	if !bytes.Equal(priv[:], priv1[:]) {
		t.Fatalf("priv create error")
	}
}

func TestAddressValid(t *testing.T) {

	fakeAddr := "1231231"
	if IsValidHexAddress(fakeAddr) {
		t.Fail()
	}

	fakeAddr = "vite_bcdc5b9dd0ed0de7de2f0e97c36638e108aa64a2bedc22c0e7"
	if IsValidHexAddress(fakeAddr) {
		t.Fail()
	}

	fakeAddr = "vite_asdc5b9dd0ed0de7de2f0e97c36638e108aa64a2bedc22c0e6"
	if IsValidHexAddress(fakeAddr) {
		t.Fail()
	}

	fakeAddr = "aite_asdc5b9dd0ed0de7de2f0e97c36638e108aa64a2bedc22c0e6"
	if IsValidHexAddress(fakeAddr) {
		t.Fail()
	}

	realAddr := "vite_bcdc5b9dd0ed0de7de2f0e97c36638e108aa64a2bedc22c0e6"
	if !IsValidHexAddress(realAddr) {
		t.Fail()
	}

}

func BenchmarkCreateAddress(b *testing.B) {
	addr, _, _ := CreateAddress()
	println(addr.Hex())
}
