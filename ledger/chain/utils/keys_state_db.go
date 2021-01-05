package chain_utils

import (
	"bytes"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
)

// -------------------------------

type StorageRealKey struct {
	len  byte
	real []byte
	cap  int
}

func (key StorageRealKey) Construct(real []byte) StorageRealKey {
	if len(real) > types.HashSize {
		panic("error key len")
	}
	realKey := StorageRealKey{}
	realKey.len = byte(len(real))

	realKey.real = make([]byte, realKey.len)
	copy(realKey.real[:], real[:realKey.len])
	realKey.cap = types.HashSize
	return realKey
}

func (key StorageRealKey) ConstructFix(all []byte) StorageRealKey {
	if len(all) != types.HashSize+1 {
		panic("error key len")
	}
	realKey := StorageRealKey{}
	realKey.len = all[len(all)-1]
	realKey.real = make([]byte, realKey.len)
	copy(realKey.real[:], all[:realKey.len])
	realKey.cap = types.HashSize
	return realKey
}

func (key StorageRealKey) Extra() []byte {
	return key.real
}

func (key StorageRealKey) String() string {
	return string(key.real)
}

// -------------------------------
type StorageKey [1 + types.AddressSize + types.HashSize + 1]byte

func (key StorageKey) Bytes() []byte {
	return key[:]
}

func (key StorageKey) String() string {
	return string(key[:])
}

func (key *StorageKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

func (key *StorageKey) KeyRefill(real StorageRealKey) {
	copy(key[1+types.AddressSize:1+types.AddressSize+types.HashSize], common.RightPadBytes(real.Extra(), types.HashSize))
	key[1+types.AddressSize+types.HashSize] = real.len
}

func (key *StorageKey) StorageKeyRefill(bytes []byte) {
	copy(key[1+types.AddressSize:1+types.AddressSize+types.HashSize], common.RightPadBytes(bytes, types.HashSize))
}

func (key *StorageKey) KeyLenRefill(len int) {
	key[1+types.AddressSize+types.HashSize] = byte(len)
}

// -------------------------------
type StorageHistoryKey [1 + types.AddressSize + types.HashSize + 1 + types.HeightSize]byte

func (key StorageHistoryKey) Bytes() []byte {
	return key[:]
}

func (key StorageHistoryKey) String() string {
	return string(key[:])
}

func (key StorageHistoryKey) Construct(bytes []byte) *StorageHistoryKey {
	if len(bytes) != 1+types.AddressSize+types.HashSize+1+types.HeightSize {
		return nil
	}
	result := &StorageHistoryKey{}
	copy(result[:], bytes[:])
	return result
}

func (key *StorageHistoryKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

func (key *StorageHistoryKey) KeyRefill(real StorageRealKey) {
	copy(key[1+types.AddressSize:1+types.AddressSize+types.HashSize], common.RightPadBytes(real.Extra(), types.HashSize))
	key[1+types.AddressSize+types.HashSize] = real.len
}

func (key *StorageHistoryKey) HeightRefill(height uint64) {
	Uint64Put(key[1+types.AddressSize+types.HashSize+1:1+types.AddressSize+types.HashSize+1+types.HeightSize], height)
}

func (key StorageHistoryKey) ExtraKeyAndLen() []byte {
	return key[1+types.AddressSize : 1+types.AddressSize+types.HashSize+1]
}

func (key StorageHistoryKey) ExtraAddress() types.Address {
	addr := types.Address{}
	addr.SetBytes(key[1 : 1+types.AddressSize])
	return addr
}
func (key StorageHistoryKey) ExtraHeight() uint64 {
	return BytesToUint64(key[1+types.AddressSize+types.HashSize+1 : 1+types.AddressSize+types.HashSize+1+types.HeightSize])
}

// -------------------------------
type BalanceKey [1 + types.AddressSize + types.TokenTypeIdSize]byte

func (key BalanceKey) Bytes() []byte {
	return key[:]
}

func (key BalanceKey) String() string {
	return string(key[:])
}

func (key *BalanceKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

func (key *BalanceKey) TokenIdRefill(tokenId types.TokenTypeId) {
	copy(key[1+types.AddressSize:1+types.AddressSize+types.TokenTypeIdSize], tokenId.Bytes())
}

// -------------------------------
type BalanceHistoryKey [1 + types.AddressSize + types.TokenTypeIdSize + types.HeightSize]byte

func (key BalanceHistoryKey) Bytes() []byte {
	return key[:]
}

func (key BalanceHistoryKey) String() string {
	return string(key[:])
}

func (key BalanceHistoryKey) Construct(bytes []byte) *BalanceHistoryKey {
	if len(bytes) != 1+types.AddressSize+types.TokenTypeIdSize+types.HeightSize {
		return nil
	}
	result := &BalanceHistoryKey{}
	copy(result[:], bytes[:])
	return result
}

func (key *BalanceHistoryKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

func (key *BalanceHistoryKey) TokenIdRefill(tokenId types.TokenTypeId) {
	copy(key[1+types.AddressSize:1+types.AddressSize+types.TokenTypeIdSize], tokenId.Bytes())
}

func (key *BalanceHistoryKey) HeightRefill(height uint64) {
	Uint64Put(key[1+types.AddressSize+types.TokenTypeIdSize:1+types.AddressSize+types.TokenTypeIdSize+types.HeightSize], height)
}

func (key BalanceHistoryKey) EqualAddressAndTokenId(addr types.Address, tokenId types.TokenTypeId) bool {
	return bytes.Equal(key[1:1+types.AddressSize], addr.Bytes()) &&
		bytes.Equal(key[1+types.AddressSize:1+types.AddressSize+types.TokenTypeIdSize], tokenId.Bytes())
}

func (key BalanceHistoryKey) ExtraTokenId() types.TokenTypeId {
	tokenId := types.TokenTypeId{}
	tokenId.SetBytes(key[1+types.AddressSize : 1+types.AddressSize+types.TokenTypeIdSize])
	return tokenId
}

func (key BalanceHistoryKey) ExtraHeight() uint64 {
	return BytesToUint64(key[1+types.AddressSize+types.TokenTypeIdSize : 1+types.AddressSize+types.TokenTypeIdSize+types.HeightSize])
}

// -------------------------------
type CodeKey [1 + types.AddressSize]byte

func (key CodeKey) Bytes() []byte {
	return key[:]
}

func (key CodeKey) String() string {
	return string(key[:])
}

func (key *CodeKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

// -------------------------------
type ContractMetaKey [1 + types.AddressSize]byte

func (key ContractMetaKey) Bytes() []byte {
	return key[:]
}

func (key ContractMetaKey) String() string {
	return string(key[:])
}

func (key *ContractMetaKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

// -------------------------------
type GidContractKey [1 + types.GidSize + types.AddressSize]byte

func (key GidContractKey) Bytes() []byte {
	return key[:]
}

func (key GidContractKey) String() string {
	return string(key[:])
}

func (key *GidContractKey) GidRefill(gid types.Gid) {
	copy(key[1:1+types.GidSize], gid.Bytes())
}

func (key *GidContractKey) AddressRefill(addr types.Address) {
	copy(key[1+types.GidSize:1+types.GidSize+types.AddressSize], addr.Bytes())
}

// -------------------------------
type VmLogListKey [1 + types.HashSize]byte

func (key VmLogListKey) Bytes() []byte {
	return key[:]
}

func (key VmLogListKey) String() string {
	return string(key[:])
}

func (key *VmLogListKey) HashRefill(hash types.Hash) {
	copy(key[1:1+types.HashSize], hash.Bytes())
}

// -------------------------------
type CallDepthKey [1 + types.HashSize]byte

func (key CallDepthKey) Bytes() []byte {
	return key[:]
}

func (key CallDepthKey) String() string {
	return string(key[:])
}

func (key *CallDepthKey) HashRefill(hash types.Hash) {
	copy(key[1:1+types.HashSize], hash.Bytes())
}
