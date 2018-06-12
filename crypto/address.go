package crypto

import (
	"encoding/hex"
	"bytes"
)

const (
	AddressSize            = 55
	PubkeyHashSize         = 20
	PubkeyHashChecksumSize = 5
	AddressPrefix          = "vite_"
)

type Address [AddressSize]byte

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func isValidAddress(address []byte) bool {

	if address == nil || len(address) != AddressSize {
		return false
	}
	if !bytes.Equal(address[:5], []byte(AddressPrefix)) {
		return false
	}

	addr := BytesToAddress(address)

	pubKeyHash, err := getPubkeyHash(addr)
	if err != nil {
		return false
	}

	pubKeyHashChecksum, err := getPubKeyHashChecksum(addr)
	if err != nil {
		return false
	}

	pubKeyHashCalc := Hash(PubkeyHashChecksumSize, pubKeyHash[:])

	if !bytes.Equal(pubKeyHashCalc, pubKeyHashChecksum[:]) {
		return false
	}

	return true
}

func PubkeyToAddress(pubkey []byte) Address {

	var addr Address
	len := copy(addr[:], AddressPrefix)

	pubkeyHash := Hash(PubkeyHashSize, pubkey)
	hexPubkeyHash := make([]byte, 2*PubkeyHashSize)
	hex.Encode(hexPubkeyHash, pubkeyHash)
	len += copy(addr[len:], hexPubkeyHash)

	pubkeyHashChecksum := Hash(PubkeyHashChecksumSize, pubkeyHash)
	hexPubkeyHashChecksum := make([]byte, 2*PubkeyHashChecksumSize)
	hex.Encode(hexPubkeyHashChecksum, pubkeyHashChecksum)
	copy(addr[len:], hexPubkeyHashChecksum)

	return addr

}

func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressSize:]
	}
	copy(a[AddressSize-len(b):], b)
}

func (addr Address) Str() (string) { return string(addr[:]) }

func CreateRandomAddress() (Address, string, error) {
	pub, pri, error := GenerateKey()
	return PubkeyToAddress(pub), hex.EncodeToString(pri), error
}

func getPubkeyHash(address Address) ([PubkeyHashSize]byte, error) {
	var b [PubkeyHashSize]byte
	_, err := hex.Decode(b[:], address[5:2*PubkeyHashSize+5])
	return b, err
}

func getPubKeyHashChecksum(address Address) ([PubkeyHashChecksumSize]byte, error) {
	var b [PubkeyHashChecksumSize]byte
	_, err := hex.Decode(b[:], address[2*PubkeyHashSize+5:])
	return b, err
}
