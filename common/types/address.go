package types

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/vitelabs/go-vite/common/helper"
	vcrypto "github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

const (
	AddressPrefix       = "vite_"
	AddressSize         = 20
	addressChecksumSize = 5
	addressPrefixLen    = len(AddressPrefix)
	hexAddressLength    = addressPrefixLen + 2*AddressSize + 2*addressChecksumSize
)

var (
	AddressPledge, _         = BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
	AddressConsensusGroup, _ = BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4})
	AddressMintage, _        = BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5})

	BuiltinContractAddrList             = []Address{AddressPledge, AddressConsensusGroup, AddressMintage}
	BuiltinContractWithoutQuotaAddrList = []Address{AddressPledge, AddressConsensusGroup, AddressMintage}
	BuiltinContractWithSendConfirm      = []Address{AddressPledge, AddressConsensusGroup, AddressMintage}
)

func IsBuiltinContractAddr(addr Address) bool {
	addrBytes := addr.Bytes()
	if helper.AllZero(addrBytes[:AddressSize-1]) && addrBytes[AddressSize-1] != byte(0) {
		return true
	}
	return false
}
func IsBuiltinContractAddrInUse(addr Address) bool {
	for _, cAddr := range BuiltinContractAddrList {
		if cAddr == addr {
			return true
		}
	}
	return false
}

func IsBuiltinContractAddrInUseWithoutQuota(addr Address) bool {
	for _, cAddr := range BuiltinContractWithoutQuotaAddrList {
		if cAddr == addr {
			return true
		}
	}
	return false
}

func IsBuiltinContractAddrInUseWithSendConfirm(addr Address) bool {
	for _, cAddr := range BuiltinContractWithSendConfirm {
		if cAddr == addr {
			return true
		}
	}
	return false
}

type Address [AddressSize]byte

var ZERO_ADDRESS = Address{}

func BytesToAddress(b []byte) (Address, error) {
	var a Address
	err := a.SetBytes(b)
	return a, err
}

func BigToAddress(b *big.Int) (Address, error) {
	return BytesToAddress(helper.LeftPadBytes(b.Bytes(), AddressSize))
}

func HexToAddress(hexStr string) (Address, error) {
	if IsValidHexAddress(hexStr) {
		addr, _ := getAddressFromHex(hexStr)
		return addr, nil
	} else {
		return Address{}, fmt.Errorf("not valid hex address %v", hexStr)
	}
}

func HexToAddressPanic(hexstr string) Address {
	h, err := HexToAddress(hexstr)
	if err != nil {
		panic(err)
	}
	return h
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
	addr, _ := BytesToAddress(vcrypto.Hash(AddressSize, pubkey))
	return addr
}

func PrikeyToAddress(key ed25519.PrivateKey) Address {
	return PubkeyToAddress(key.PubByte())
}

func (addr *Address) SetBytes(b []byte) error {
	if length := len(b); length != AddressSize {
		return fmt.Errorf("error address size  %v", length)
	}
	copy(addr[:], b)
	return nil
}

func (addr Address) Hex() string {
	return AddressPrefix + hex.EncodeToString(addr[:]) + hex.EncodeToString(vcrypto.Hash(addressChecksumSize, addr[:]))
}
func (addr Address) Bytes() []byte { return addr[:] }
func (addr Address) String() string {
	return addr.Hex()
}

func CreateAddress() (Address, ed25519.PrivateKey, error) {
	pub, pri, error := ed25519.GenerateKey(rand.Reader)
	return PubkeyToAddress(pub), pri, error
}

func CreateContractAddress(data ...[]byte) Address {
	addr, _ := BytesToAddress(vcrypto.Hash(AddressSize, data...))
	return addr
}

func CreateAddressWithDeterministic(d [32]byte) (Address, ed25519.PrivateKey, error) {
	pub, pri, error := ed25519.GenerateKeyFromD(d)
	return PubkeyToAddress(pub), pri, error
}

func getAddressFromHex(hexStr string) ([AddressSize]byte, error) {
	var b [AddressSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[addressPrefixLen:2*AddressSize+addressPrefixLen]))
	return b, err
}

func getAddressChecksumFromHex(hexStr string) ([addressChecksumSize]byte, error) {
	var b [addressChecksumSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[2*AddressSize+addressPrefixLen:]))
	return b, err
}

func (a *Address) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return ErrJsonNotString
	}
	addresses, e := HexToAddress(string(trimLeftRightQuotation(input)))
	if e != nil {
		return e
	}
	a.SetBytes(addresses.Bytes())
	return nil
}

func (a Address) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}
