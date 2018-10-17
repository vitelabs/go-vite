package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"github.com/stretchr/testify/assert"
	"encoding/hex"
)

func TestCreateContractAddress(t *testing.T) {
	addr := CreateContractAddress([]byte{1, 2, 3}, []byte{1, 2, 3})
	fmt.Println(addr)
	if !IsValidHexAddress(addr.String()) {
		t.Fatal("Not valid")
	}
}

func TestCreateRandomAddress(t *testing.T) {
	for i := 0; i < 20; i++ {
		addr, priv, _ := CreateAddress()
		fmt.Println(addr, hex.EncodeToString(priv))
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
