package common

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	vcrypto "github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"strings"
)

const (
	AddressPrefix       = "vite_"
	addressSize         = 20
	addressChecksumSize = 5
	addressPrefixLen    = len(AddressPrefix)
	hexAddressLength    = addressPrefixLen + 2*addressSize + 2*addressChecksumSize
)

type Address [addressSize]byte

func BytesToAddress(b []byte) (Address, error) {
	var a Address
	err := a.SetBytes(b)
	return a, err
}

func HexToAddress(hexStr string) (Address, error) {
	if IsValidHexAddress(hexStr) {
		addr, _ := getAddressFromHex(hexStr)
		return addr, nil
	} else {
		return Address{}, fmt.Errorf("Not valid hex Address")
	}
}

func IsValidHexAddress(hexStr string) bool {
	if len(hexStr) != hexAddressLength || !strings.HasPrefix(hexStr, AddressPrefix) {
		return false
	}

	address, err := getAddressFromHex(hexStr)
	if err != nil {
		return false
	}

	addressChecksum, err := getAddressChecksumFromHex(hexStr)
	if err != nil {
		return false
	}

	if !bytes.Equal(vcrypto.Hash(addressChecksumSize, address[:]), addressChecksum[:]) {
		return false

	}

	return true
}

func PubkeyToAddress(pubkey []byte) Address {
	addr, _ := BytesToAddress(vcrypto.Hash(addressSize, pubkey))
	return addr
}

func PrikeyToAddress(key ed25519.PrivateKey) Address {
	return PubkeyToAddress(key.PubByte())
}

func (a *Address) SetBytes(b []byte) error {
	if length := len(b); length != addressSize {
		return fmt.Errorf("address bytes length error %v", length)
	}
	copy(a[:], b)
	return nil
}

func (addr Address) Hex() string {
	return AddressPrefix + hex.EncodeToString(addr[:]) + hex.EncodeToString(vcrypto.Hash(addressChecksumSize, addr[:]))
}
func (addr Address) Bytes() []byte { return addr[:] }
func (a Address) String() string {
	return a.Hex()
}

func CreateAddress() (Address, ed25519.PrivateKey, error) {
	pub, pri, error := ed25519.GenerateKey(rand.Reader)
	return PubkeyToAddress(pub), pri, error
}

func CreateAddressWithDeterministic(d [32]byte) (Address, ed25519.PrivateKey, error) {
	pub, pri, error := ed25519.GenerateKeyFromD(d)
	return PubkeyToAddress(pub), pri, error
}

func getAddressFromHex(hexStr string) ([addressSize]byte, error) {
	var b [addressSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[addressPrefixLen:2*addressSize+addressPrefixLen]))
	return b, err
}

func getAddressChecksumFromHex(hexStr string) ([addressChecksumSize]byte, error) {
	var b [addressChecksumSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[2*addressSize+addressPrefixLen:]))
	return b, err
}
