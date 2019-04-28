package types

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/go-errors/errors"

	"github.com/vitelabs/go-vite/common/helper"
	vcrypto "github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

const (
	AddressPrefix       = "vite_"
	AddressSize         = 21
	AddressCoreSize     = 20
	addressChecksumSize = 5
	addressPrefixLen    = len(AddressPrefix)
	hexAddrCoreLen      = 2 * AddressCoreSize
	hexAddrChecksumLen  = 2 * addressChecksumSize
	hexAddressLength    = addressPrefixLen + hexAddrCoreLen + hexAddrChecksumLen
)

const (
	UserAddrByte     = byte(0)
	ContractAddrByte = byte(1)
)

var (
	AddressPledge, _         = BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, ContractAddrByte})
	AddressConsensusGroup, _ = BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, ContractAddrByte})
	AddressMintage, _        = BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, ContractAddrByte})

	BuiltinContractAddrList             = []Address{AddressPledge, AddressConsensusGroup, AddressMintage}
	BuiltinContractWithoutQuotaAddrList = []Address{AddressPledge, AddressConsensusGroup, AddressMintage}
	BuiltinContractWithSendConfirm      = []Address{AddressPledge, AddressConsensusGroup, AddressMintage}
)

func IsContractAddr(addr Address) bool {
	return addr[AddressSize-1] == ContractAddrByte
}

func IsBuiltinContractAddr(addr Address) bool {
	addrBytes := addr.Bytes()
	if IsContractAddr(addr) && helper.AllZero(addrBytes[:AddressCoreSize-1]) && addrBytes[AddressCoreSize-1] != byte(0) {
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
	return ValidHexAddress(hexStr)
}

func HexToAddressPanic(hexstr string) Address {
	h, err := HexToAddress(hexstr)
	if err != nil {
		panic(err)
	}
	return h
}

func IsValidHexAddress(hexStr string) bool {
	_, err := ValidHexAddress(hexStr)
	return err == nil
}

func ValidHexAddress(hexStr string) (Address, error) {
	if len(hexStr) != hexAddressLength || !strings.HasPrefix(hexStr, AddressPrefix) {
		return Address{}, errors.New("error hex address or prefix")
	}

	addrCore, err := getAddrCoreFromHex(hexStr)
	if err != nil {
		return Address{}, err
	}

	addressChecksum, err := getAddressChecksumFromHex(hexStr)
	if err != nil {
		return Address{}, err
	}

	checksum := vcrypto.Hash(addressChecksumSize, addrCore[:])

	if bytes.Equal(checksum, addressChecksum[:]) {
		return GenUserAddress(addrCore[:])
	}

	if bytes.Equal(checksum, helper.LDI(addressChecksum[:])) {
		return GenContractAddress(addrCore[:])
	}
	return Address{}, errors.Errorf("error address[%s] checksum", hexStr)
}

func PubkeyToAddress(pubkey []byte) Address {
	hash := vcrypto.Hash(AddressCoreSize, pubkey)
	addr, err := GenUserAddress(hash)
	if err != nil {
		panic(err)
	}
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
	coreAddr := addr[:AddressCoreSize]
	byt := addr[AddressCoreSize]
	if byt == UserAddrByte {
		return AddressPrefix + hex.EncodeToString(coreAddr) + hex.EncodeToString(vcrypto.Hash(addressChecksumSize, coreAddr))
	} else if byt == ContractAddrByte {
		hash := vcrypto.Hash(addressChecksumSize, coreAddr)
		return AddressPrefix + hex.EncodeToString(coreAddr) + hex.EncodeToString(helper.LDI(hash))
	} else {
		return fmt.Sprintf("error address[%d]", byt)
	}
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
	addr, _ := BytesToAddress(append(vcrypto.Hash(AddressCoreSize, data...), ContractAddrByte))
	return addr
}

func GenContractAddress(data []byte) (Address, error) {
	var addr []byte
	addr = append(addr, data[:]...)
	addr = append(addr, ContractAddrByte)
	return BytesToAddress(addr[:])
}

func GenUserAddress(data []byte) (Address, error) {
	var addr []byte
	addr = append(addr, data[:]...)
	addr = append(addr, UserAddrByte)
	return BytesToAddress(addr[:])
}

func CreateAddressWithDeterministic(d [32]byte) (Address, ed25519.PrivateKey, error) {
	pub, pri, error := ed25519.GenerateKeyFromD(d)
	return PubkeyToAddress(pub), pri, error
}

func getAddressFromHex(hexStr string) ([AddressSize]byte, error) {
	var b [AddressSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[addressPrefixLen:hexAddrCoreLen+addressPrefixLen]))
	return b, err
}

func getAddrCoreFromHex(hexStr string) ([AddressCoreSize]byte, error) {
	var b [AddressCoreSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[addressPrefixLen:hexAddrCoreLen+addressPrefixLen]))
	return b, err
}

func getAddressChecksumFromHex(hexStr string) ([addressChecksumSize]byte, error) {
	var b [addressChecksumSize]byte
	_, err := hex.Decode(b[:], []byte(hexStr[hexAddrCoreLen+addressPrefixLen:]))
	return b, err
}

func (a *Address) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return ErrJsonNotString
	}

	addresses, e := HexToAddress(strings.Trim(string(input), "\""))
	if e != nil {
		return e
	}
	a.SetBytes(addresses.Bytes())
	return nil
}

func (a Address) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}
