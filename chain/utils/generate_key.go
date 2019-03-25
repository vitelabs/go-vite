package chain_utils

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/types"
)

// ====== index db ======
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
	key = append(key, Uint64ToBytes(height)...)
	return key
}

func CreateReceiveKey(sendBlockHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(key, ReceiveKeyPrefix)
	key = append(key, sendBlockHash.Bytes()...)
	return key
}

func CreateConfirmHeightKey(addr *types.Address, height uint64) []byte {
	key := make([]byte, 0, 1+types.AddressSize+8)
	key = append(key, ConfirmHeightKeyPrefix)
	key = append(key, addr.Bytes()...)
	key = append(key, Uint64ToBytes(height)...)
	return key
}

func CreateAccountAddressKey(addr *types.Address) []byte {
	addrBytes := addr.Bytes()
	key := make([]byte, 0, 1+types.AddressSize)
	key = append(key, AccountAddressKeyPrefix)
	key = append(key, addrBytes...)
	return key
}

func CreateOnRoadKey(addr *types.Address, id uint64) []byte {
	key := make([]byte, 0, 1+types.AddressSize+8)
	key = append(key, OnRoadKeyPrefix)
	key = append(key, addr.Bytes()...)
	key = append(key, Uint64ToBytes(id)...)
	return key
}

func CreateOnRoadPrefixKey(addr *types.Address) []byte {
	key := make([]byte, 0, 1+types.AddressSize)
	key = append(key, OnRoadKeyPrefix)
	key = append(key, addr.Bytes()...)
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

func CreateAccountIdKey(accountId uint64) []byte {
	key := make([]byte, 0, 9)
	key = append(key, AccountIdKeyPrefix)
	key = append(key, Uint64ToBytes(accountId)...)

	return key
}

func CreateAccountIdPrefixKey() []byte {
	return []byte{AccountIdKeyPrefix}
}

func CreateAccountBlockHeightPrefixKey(addr *types.Address) []byte {
	key := make([]byte, 0, 1+types.AddressSize)

	key = append(key, AccountBlockHeightKeyPrefix)
	key = append(key, addr.Bytes()...)
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
	key = append(key, Uint64ToBytes(snapshotBlockHeight)...)
	return key
}

func CreateIndexDbLatestLocationKey() []byte {
	return []byte{IndexDbLatestLocationKeyPrefix}
}

// ====== state db ======

func CreateStorageValueKeyPrefix(address *types.Address, prefix []byte) []byte {
	keySize := 1 + types.AddressSize + len(prefix)
	key := make([]byte, 0, keySize)

	key = append(key, StorageKeyPrefix)
	key = append(key, address.Bytes()...)
	key = append(key, prefix...)
	return key
}

func CreateStorageValueKey(address *types.Address, storageKey []byte) []byte {
	keySize := types.AddressSize + 34
	key := make([]byte, keySize)
	key[0] = StorageKeyPrefix

	copy(key[1:types.AddressSize+1], address.Bytes())
	copy(key[types.AddressSize+1:], storageKey)
	key[keySize-1] = byte(len(storageKey))

	return key
}

func CreateHistoryStorageValueKey(address *types.Address, storageKey []byte, snapshotHeight uint64) []byte {
	keySize := types.AddressSize + 42
	key := make([]byte, keySize)
	key[0] = StorageHistoryKeyPrefix

	copy(key[1:types.AddressSize+1], address.Bytes())
	copy(key[types.AddressSize+1:], storageKey)
	key[keySize-8] = byte(len(storageKey))
	binary.BigEndian.PutUint64(key[keySize-7:], snapshotHeight)

	return key
}

func CreateHistoryStorageValueKeyPrefix(address *types.Address, prefix []byte) []byte {
	keySize := 1 + types.AddressSize + len(prefix)
	key := make([]byte, 0, keySize)

	key = append(key, StorageHistoryKeyPrefix)
	key = append(key, address.Bytes()...)
	key = append(key, prefix...)
	return key
}

func CreateBalanceKey(address *types.Address, tokenTypeId *types.TokenTypeId) []byte {
	key := make([]byte, 1+types.AddressSize+types.TokenTypeIdSize)
	key[0] = BalanceKeyPrefix

	copy(key[1:types.AddressSize+1], address.Bytes())
	copy(key[types.AddressSize+1:], tokenTypeId.Bytes())

	return key
}

func CreateHistoryBalanceKey(address *types.Address, tokenTypeId *types.TokenTypeId, snapshotHeight uint64) []byte {
	keySize := 1 + types.AddressSize + types.TokenTypeIdSize + 8

	key := make([]byte, keySize)

	key[0] = BalanceHistoryKeyPrefix

	copy(key[1:types.AddressSize+1], address.Bytes())

	copy(key[types.AddressSize+1:], tokenTypeId.Bytes())

	binary.BigEndian.PutUint64(key[keySize-7:], snapshotHeight)

	return key
}

func CreateCodeKey(address *types.Address) []byte {
	keySize := 1 + types.AddressSize

	key := make([]byte, keySize)

	key[0] = CodeKeyPrefix

	copy(key[1:], address.Bytes())

	return key
}

func CreateContractMetaKey(address *types.Address) []byte {
	keySize := 1 + types.AddressSize

	key := make([]byte, keySize)

	key[0] = ContractMetaKeyPrefix

	copy(key[1:], address.Bytes())

	return key
}
func CreateGidContractKey(gid *types.Gid, address *types.Address) []byte {
	key := make([]byte, 0, 1+types.GidSize+types.AddressSize)

	key[0] = GidContractKeyPrefix

	key = append(key, gid.Bytes()...)
	key = append(key, address.Bytes()...)

	return key
}

func CreateGidContractPrefixKey(gid *types.Gid) []byte {
	key := make([]byte, 0, 1+types.GidSize)

	key[0] = GidContractKeyPrefix

	key = append(key, gid.Bytes()...)

	return key
}

func CreateVmLogListKey(logHash *types.Hash) []byte {
	key := make([]byte, 1+types.HashSize)

	key[0] = VmLogListKeyPrefix

	copy(key[1:], logHash.Bytes())

	return key
}

func CreateCallDepthKey(blockHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(key, CallDepthKeyPrefix)
	key = append(key, blockHash.Bytes()...)
	return key
}

func CreateUndoLocationKey() []byte {
	return []byte{UndoLocationKeyPrefix}
}

func CreateStateDbLocationKey() []byte {
	return []byte{StateDbLocationKeyPrefix}
}

//// ====== state_bak db ======
//
//func CreateBalanceKey(addr *types.Address, tokenTypeId *types.TokenTypeId) []byte {
//	key := make([]byte, 0, 1+types.AddressSize+types.TokenTypeIdSize)
//	key = append(key, BalanceKeyPrefix)
//	key = append(key, addr.Bytes()...)
//	key = append(key, tokenTypeId.Bytes()...)
//	return key
//}
//
//func CreateStorageKeyPrefix(addr *types.Address, storageKey []byte) []byte {
//	key := make([]byte, 0, 1+types.AddressSize+len(storageKey))
//
//	key = append(key, StorageKeyPrefix)
//	key = append(key, addr.Bytes()...)
//	key = append(key, storageKey...)
//	return key
//}
//
//func CreateCodeKey(addr *types.Address) []byte {
//	key := make([]byte, 0, 1+types.AddressSize)
//
//	key = append(key, CodeKeyPrefix)
//	key = append(key, addr.Bytes()...)
//	return key
//}
//
//func CreateContractMetaKey(addr *types.Address) []byte {
//	key := make([]byte, 0, 1+types.AddressSize)
//	key = append(key, ContractMetaKeyPrefix)
//	key = append(key, addr.Bytes()...)
//	return key
//}
//
//func CreateVmLogListKey(logHash *types.Hash) []byte {
//	key := make([]byte, 0, 1+types.HashSize)
//	key = append(key, VmLogListKeyPrefix)
//	key = append(key, logHash.Bytes()...)
//	return key
//}
//
//// ====== mv db ======
//
//func CreateKeyIdKey(mvDbKey []byte) []byte {
//	key := make([]byte, 0, 1+len(mvDbKey))
//	key = append(key, KeyIdKeyPrefix)
//	key = append(key, mvDbKey...)
//	return key
//}
//
//func CreateValueIdKey(valueId uint64) []byte {
//	key := make([]byte, 0, 9)
//	key = append(key, ValueIdKeyPrefix)
//	key = append(key, Uint64ToBytes(valueId)...)
//	return key
//}
//
//func CreateLatestValueKey(keyId uint64) []byte {
//	key := make([]byte, 0, 9)
//	key = append(key, LatestValueKeyPrefix)
//	key = append(key, Uint64ToBytes(keyId)...)
//	return key
//}
//
//func CreateUndoKey(blockHash *types.Hash) []byte {
//	key := make([]byte, 0, 33)
//	key = append(key, UndoKeyPrefix)
//	key = append(key, blockHash.Bytes()...)
//	return key
//}
//
//func CreateMvDbLatestLocationKey() []byte {
//	return []byte{MvDbLatestLocationKeyPrefix}
//}
//
//func CreateStorageValueKey(address *types.Address, storageKey []byte) []byte {
//	keySize := types.AddressSize + 34
//	key := make([]byte, keySize)
//	key[0] = StorageKeyPrefix
//
//	copy(key[1:types.AddressSize+1], address.Bytes())
//	copy(key[types.AddressSize+1:], storageKey)
//	key[keySize-1] = byte(len(storageKey))
//
//	return key
//}
//
//func CreateHistoryStorageValueKey(address *types.Address, storageKey []byte, snapshotHeight uint64) []byte {
//	keySize := types.AddressSize + 42
//	key := make([]byte, keySize)
//	key[0] = StorageHistoryKeyPrefix
//
//	copy(key[1:types.AddressSize+1], address.Bytes())
//	copy(key[types.AddressSize+1:], storageKey)
//	key[keySize-8] = byte(len(storageKey))
//	binary.BigEndian.PutUint64(key[keySize-7:], snapshotHeight)
//
//	return key
//}
