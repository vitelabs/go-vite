package common

import (
	"encoding/hex"
	"crypto/rand"
	"go-vite/crypto/ed25519"
	hash "go-vite/crypto"
	"strings"
	"bytes"
)

const (
	AddressSize         = 20
	AddressChecksumSize = 5
	AddressPrefix       = "vite_"
	AddressPrefixLen    = len(AddressPrefix)
	HexAddressLength    = AddressPrefixLen + 2*AddressSize + 2*AddressChecksumSize
)

type Address [AddressSize]byte

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func IsValidHexAddress(hexStr string) bool {
	if len(hexStr) != HexAddressLength || !strings.HasPrefix(hexStr, AddressPrefix) {
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

	if !bytes.Equal(hash.Hash(AddressChecksumSize, address[:]), addressChecksum[:]) {
		return false

	}

	return true
}



func PubkeyToAddress(pubkey []byte) Address {
	return BytesToAddress(hash.Hash(AddressSize, pubkey))
}

func PrikeyToAddress(key ed25519.PrivateKey) Address {
	return PubkeyToAddress(key.PubByte())
}

func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressSize:]
	}
	copy(a[AddressSize-len(b):], b)
}

func (addr Address) Hex() string   {
	return AddressPrefix + hex.EncodeToString(addr[:]) + hex.EncodeToString(hash.Hash(AddressChecksumSize, addr[:]))
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

func getAddressFromHex(hexStr string) ([AddressSize]byte, error) {
	var b [AddressSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[AddressPrefixLen:2*AddressSize+AddressPrefixLen]))
	return b, err
}

func getAddressChecksumFromHex(hexStr string) ([AddressChecksumSize]byte, error) {
	var b [AddressChecksumSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[2*AddressSize+AddressPrefixLen:]))
	return b, err
}
