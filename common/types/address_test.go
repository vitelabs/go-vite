package types

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

func TestCreateContractAddress(t *testing.T) {
	addr := CreateContractAddress([]byte{1, 2, 3}, []byte{1, 2, 3})
	fmt.Println(addr)
	if _, err := ValidHexAddress(addr.String()); err == nil {
		t.Fatal("Not valid")
	}
}

func TestCreateRandomAddress(t *testing.T) {
	for i := 0; i < 10; i++ {
		_, priv, _ := CreateAddress()
		fmt.Println("\"" + hex.EncodeToString(priv) + "\",")
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

	real, _, err := CreateAddress()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !IsValidHexAddress(real.String()) {
		t.Fail()
	}

	if !IsValidHexAddress(AddressPledge.String()) {
		t.Fail()
	}

	if !IsValidHexAddress(AddressConsensusGroup.String()) {
		t.Fail()
	}

	if !IsValidHexAddress(AddressMintage.String()) {
		t.Fail()
	}

}

func BenchmarkCreateAddress(b *testing.B) {
	addr, _, _ := CreateAddress()
	println(addr.Hex())
}

func TestAddress_UnmarshalJSON(t *testing.T) {
	addr0, _, _ := CreateAddress()
	marshal, _ := json.Marshal(addr0)
	var addr Address
	e := json.Unmarshal([]byte(marshal), &addr)
	if e != nil {
		t.Fatal(e)
	}
	assert.Equal(t, addr0.String(), addr.String())
}

func TestPubkeyToAddress(t *testing.T) {
	byt, err := base64.StdEncoding.DecodeString("meHN+pdEEN1yp34IV8JZRFYqYMB+znhxvSTMRufmeoc=")
	if err != nil {
		panic(err)
	}
	publicKey, err := ed25519.HexToPublicKey(hex.EncodeToString(byt))
	if err != nil {
		panic(err)
	}

	producer := PubkeyToAddress(publicKey)

	fmt.Printf("%+v\n", producer)
}

func TestPubkeyBytesToAddress(t *testing.T) {
	byt := []byte{63, 197, 34, 78, 89, 67, 59, 255, 79, 72, 200, 60, 14, 180, 237, 234, 14, 76, 66, 234, 105, 126, 4, 205, 236, 113, 125, 3, 229, 13, 82, 0}
	producer := PubkeyToAddress(byt)
	t.Log(producer)
}
