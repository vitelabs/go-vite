package chain_dbutils

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

func CreateAccountIdKey(accountId uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, AccountIdKeyPrefix)
	key = append(key, Uint64ToFixedBytes(accountId)...)

	return accountIdKey
}

func CreateConfirmHeightKey(accountId, height uint64) []byte {
	key := make([]byte, 0, 17)
	key = append(key, ConfirmHeightKeyPrefix)
	key = append(key, Uint64ToFixedBytes(accountId)...)
	key = append(key, Uint64ToFixedBytes(height)...)
	return key
}
func CreateAccountBlockHashKey(blockHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(key, AccountBlockHashKeyPrefix)
	key = append(key, blockHash.Bytes()...)
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

func CreateValueIdKey(valueId uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, ValueIdKeyPrefix)
	key = append(key, Uint64ToFixedBytes(valueId)...)
	return key
}

func CreateStorageKeyKey(accountId uint64, storageKey []byte) []byte {
	key := make([]byte, 0, 9+len(storageKey))
	key = append(key, StorageKeyKeyPrefix)
	key = append(key, Uint64ToFixedBytes(accountId)...)
	key = append(key, storageKey...)
	return key
}

func CreateBalanceKey(accountId uint64, tokenTypeId *types.TokenTypeId) []byte {
	key := make([]byte, 0, 1+8+types.TokenTypeIdSize)
	key = append(key, BalanceKeyPrefix)
	key = append(key, Uint64ToFixedBytes(accountId)...)
	key = append(key, tokenTypeId.Bytes()...)
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
