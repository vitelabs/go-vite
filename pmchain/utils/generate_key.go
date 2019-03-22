package chain_utils

import (
	"github.com/vitelabs/go-vite/common/types"
)

func CreateAccountAddressKey(addr *types.Address) []byte {
	addrBytes := addr.Bytes()
	key := make([]byte, 0, 1+types.AddressSize)
	key = append(key, AccountAddressKeyPrefix)
	key = append(key, addrBytes...)
	return key
}

func CreateReceiveKey(sendBlockHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(key, ReceiveKeyPrefix)
	key = append(key, sendBlockHash.Bytes()...)
	return key
}

func CreateOnRoadKey(addr *types.Address, id uint64) []byte {
	key := make([]byte, 0, 1+types.AddressSize+8)
	key = append(key, OnRoadKeyPrefix)
	key = append(key, addr.Bytes()...)
	key = append(key, Uint64ToFixedBytes(id)...)
	return key
}

func CreateOnRoadPrefixKey(toAccountId uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, OnRoadKeyPrefix)
	key = append(key, Uint64ToFixedBytes(toAccountId)...)
	return key
}

func CreateOnRoadReverseKey(reverseKey []byte) []byte {
	key := make([]byte, 0, 1+len(reverseKey))
	key = append(key, OnRoadReverseKeyPrefix)
	key = append(key, reverseKey...)
	return key
}

func CreateLatestOnRoadIdKey() []byte {
	return []byte{LatestOnRoadIdKeyPrefix}
}

func CreateVmLogListKey(logHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(key, VmLogListKeyPrefix)
	key = append(key, logHash.Bytes()...)
	return key
}

func CreateAccountIdKey(accountId uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, AccountIdKeyPrefix)
	key = append(key, Uint64ToFixedBytes(accountId)...)

	return key
}

func CreateConfirmHeightKey(blockHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(key, ConfirmHeightKeyPrefix)
	key = append(key, blockHash.Bytes()...)
	return key
}
func CreateAccountBlockHashKey(blockHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(key, AccountBlockHashKeyPrefix)
	key = append(key, blockHash.Bytes()...)
	return key
}

func CreateAccountBlockHeightKey(addr *types.Address, height uint64) []byte {
	key := make([]byte, 0, 1+types.AddressSize+8)

	key = append(key, AccountBlockHeightKeyPrefix)
	key = append(key, addr.Bytes()...)
	key = append(key, Uint64ToFixedBytes(height)...)
	return key
}

func CreateSnapshotBlockHashKey(snapshotBlockHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(key, SnapshotBlockHashKeyPrefix)
	key = append(key, snapshotBlockHash.Bytes()...)
	return key
}

func CreateSnapshotBlockHeightKey(snapshotBlockHeight uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, SnapshotBlockHeightKeyPrefix)
	key = append(key, Uint64ToFixedBytes(snapshotBlockHeight)...)
	return key
}

func CreateKeyIdKey(mvDbKey []byte) []byte {
	key := make([]byte, 0, 1+len(mvDbKey))
	key = append(key, KeyIdKeyPrefix)
	key = append(key, mvDbKey...)
	return key
}

func CreateValueIdKey(valueId uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, ValueIdKeyPrefix)
	key = append(key, Uint64ToFixedBytes(valueId)...)
	return key
}

func CreateLatestValueKey(keyId uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, LatestValueKeyPrefix)
	key = append(key, Uint64ToFixedBytes(keyId)...)
	return key
}

func CreateBalanceKey(accountId uint64, tokenTypeId *types.TokenTypeId) []byte {
	key := make([]byte, 0, 1+8+types.TokenTypeIdSize)
	key = append(key, BalanceKeyPrefix)
	key = append(key, Uint64ToFixedBytes(accountId)...)
	key = append(key, tokenTypeId.Bytes()...)
	return key
}

func CreateStorageKeyPrefix(accountId uint64, storageKey []byte) []byte {
	key := make([]byte, 0, 1+8+len(storageKey))

	key = append(key, StorageKeyPrefix)
	key = append(key, Uint64ToFixedBytes(accountId)...)
	key = append(key, storageKey...)
	return key
}

func CreateCodeKey(accountId uint64) []byte {
	key := make([]byte, 0, 9)

	key = append(key, CodeKeyPrefix)
	key = append(key, Uint64ToFixedBytes(accountId)...)
	return key
}

func CreateContractMetaKey(accountId uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, ContractMetaKeyPrefix)
	key = append(key, Uint64ToFixedBytes(accountId)...)
	return key
}

func CreateStateUndoKey(blockHash *types.Hash) []byte {
	key := make([]byte, 0, 33)
	key = append(key, StateUndoKeyPrefix)
	key = append(key, blockHash.Bytes()...)
	return key
}
