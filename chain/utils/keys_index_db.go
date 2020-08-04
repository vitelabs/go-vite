package chain_utils

import (
	"github.com/vitelabs/go-vite/common/types"
)

// -------------------------------
type AccountBlockHashKey [1 + types.HashSize]byte

func (key AccountBlockHashKey) Bytes() []byte {
	return key[:]
}

func (key AccountBlockHashKey) String() string {
	return string(key[:])
}

func (key *AccountBlockHashKey) HashRefill(hash types.Hash) {
	copy(key[1:1+types.HashSize], hash.Bytes())
}

// -------------------------------
type AccountBlockHeightKey [1 + types.AddressSize + types.HeightSize]byte

func (key AccountBlockHeightKey) Bytes() []byte {
	return key[:]
}
func (key AccountBlockHeightKey) String() string {
	return string(key[:])
}

func (key *AccountBlockHeightKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

func (key *AccountBlockHeightKey) HeightRefill(height uint64) {
	Uint64Put(key[1+types.AddressSize:1+types.AddressSize+types.HeightSize], height)
}

// -------------------------------
type ReceiveKey [1 + types.HashSize]byte

func (key ReceiveKey) Bytes() []byte {
	return key[:]
}
func (key ReceiveKey) String() string {
	return string(key[:])
}

func (key *ReceiveKey) HashRefill(hash types.Hash) {
	copy(key[1:1+types.HashSize], hash.Bytes())
}

// --------------------------------
type ConfirmHeightKey [1 + types.AddressSize + types.HeightSize]byte

func (key ConfirmHeightKey) Bytes() []byte {
	return key[:]
}

func (key ConfirmHeightKey) String() string {
	return string(key[:])
}

func (key *ConfirmHeightKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

func (key *ConfirmHeightKey) HeightRefill(height uint64) {
	Uint64Put(key[1+types.AddressSize:1+types.AddressSize+types.HeightSize], height)
}

// --------------------------------
type OnRoadKey [1 + types.AddressSize + types.HashSize]byte

func (key OnRoadKey) Bytes() []byte {
	return key[:]
}

func (key OnRoadKey) String() string {
	return string(key[:])
}

func (key *OnRoadKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

func (key *OnRoadKey) HashRefill(hash types.Hash) {
	copy(key[1+types.AddressSize:1+types.AddressSize+types.HashSize], hash.Bytes())
}

// --------------------------------
type SnapshotBlockHashKey [1 + types.HashSize]byte

func (key SnapshotBlockHashKey) Bytes() []byte {
	return key[:]
}

func (key SnapshotBlockHashKey) String() string {
	return string(key[:])
}

func (key *SnapshotBlockHashKey) HashRefill(hash types.Hash) {
	copy(key[1:1+types.HashSize], hash.Bytes())
}

// --------------------------------
type SnapshotBlockHeightKey [1 + types.HeightSize]byte

func (key SnapshotBlockHeightKey) Bytes() []byte {
	return key[:]
}

func (key SnapshotBlockHeightKey) String() string {
	return string(key[:])
}

func (key *SnapshotBlockHeightKey) HeightRefill(height uint64) {
	Uint64Put(key[1:1+types.HeightSize], height)
}

// --------------------------------
type AccountAddressKey [1 + types.AddressSize]byte

func (key AccountAddressKey) Bytes() []byte {
	return key[:]
}

func (key AccountAddressKey) String() string {
	return string(key[:])
}

func (key *AccountAddressKey) AddressRefill(addr types.Address) {
	copy(key[1:1+types.AddressSize], addr.Bytes())
}

// --------------------------------
type AccountIdKey [1 + types.AccountIdSize]byte

func (key AccountIdKey) Bytes() []byte {
	return key[:]
}

func (key AccountIdKey) String() string {
	return string(key[:])
}

func (key *AccountIdKey) AccountIdRefill(accountId uint64) {
	Uint64Put(key[1:1+types.AccountIdSize], accountId)
}
