package chain_utils

import (
	"github.com/vitelabs/go-vite/common/types"
)

// ====== index db ======
func CreateAccountBlockHashKey(blockHash *types.Hash) AccountBlockHashKey {
	key := AccountBlockHashKey{}
	key[0] = AccountBlockHashKeyPrefix
	key.HashRefill(*blockHash)
	return key
}

func CreateAccountBlockHeightKey(addr *types.Address, height uint64) AccountBlockHeightKey {
	key := AccountBlockHeightKey{}
	key[0] = AccountBlockHeightKeyPrefix
	key.AddressRefill(*addr)
	key.HeightRefill(height)
	return key
}

func CreateReceiveKey(sendBlockHash *types.Hash) ReceiveKey {
	key := ReceiveKey{}
	key[0] = ReceiveKeyPrefix
	key.HashRefill(*sendBlockHash)
	return key
}

func CreateConfirmHeightKey(addr *types.Address, height uint64) ConfirmHeightKey {
	key := ConfirmHeightKey{}
	key[0] = ConfirmHeightKeyPrefix
	key.AddressRefill(*addr)
	key.HeightRefill(height)
	return key
}

func CreateAccountAddressKey(addr *types.Address) AccountAddressKey {
	key := AccountAddressKey{}
	key[0] = AccountAddressKeyPrefix
	key.AddressRefill(*addr)
	return key
}

func CreateOnRoadKey(toAddr types.Address, blockHash types.Hash) OnRoadKey {
	key := OnRoadKey{}
	key[0] = OnRoadKeyPrefix
	key.AddressRefill(toAddr)
	key.HashRefill(blockHash)
	return key
}

func CreateAccountIdKey(accountId uint64) AccountIdKey {
	key := AccountIdKey{}
	key[0] = AccountIdKeyPrefix
	key.AccountIdRefill(accountId)
	return key
}

func CreateSnapshotBlockHashKey(snapshotBlockHash *types.Hash) SnapshotBlockHashKey {
	key := SnapshotBlockHashKey{}
	key[0] = SnapshotBlockHashKeyPrefix
	key.HashRefill(*snapshotBlockHash)
	return key
}

func CreateSnapshotBlockHeightKey(snapshotBlockHeight uint64) SnapshotBlockHeightKey {
	key := SnapshotBlockHeightKey{}
	key[0] = SnapshotBlockHeightKeyPrefix
	key.HeightRefill(snapshotBlockHeight)
	return key
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

func CreateStorageValueKey(address *types.Address, storageKey []byte) StorageKey {
	if len(storageKey) > types.HashSize {
		panic("error storage key len.")
	}
	key := StorageKey{}
	key[0] = StorageKeyPrefix
	key.AddressRefill(*address)
	key.KeyRefill(StorageRealKey{}.Construct(storageKey))
	//key.StorageKeyRefill(storageKey)
	//key.KeyLenRefill(len(storageKey))
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

func CreateHistoryStorageValueKey(address *types.Address, storageKey []byte, snapshotHeight uint64) StorageHistoryKey {
	if len(storageKey) > types.HashSize {
		panic("error history storage key len.")
	}
	key := StorageHistoryKey{}
	key[0] = StorageHistoryKeyPrefix
	key.AddressRefill(*address)
	key.KeyRefill(StorageRealKey{}.Construct(storageKey))
	//key.StorageKeyRefill(storageKey)
	//key.KeyLenRefill(len(storageKey))
	key.HeightRefill(snapshotHeight)
	return key
}

func CreateBalanceKeyPrefix(address types.Address) []byte {
	key := make([]byte, 1+types.AddressSize)
	key[0] = BalanceKeyPrefix
	copy(key[1:types.AddressSize+1], address.Bytes())

	return key
}

func CreateBalanceKey(address types.Address, tokenTypeId types.TokenTypeId) BalanceKey {
	key := BalanceKey{}
	key[0] = BalanceKeyPrefix
	key.AddressRefill(address)
	key.TokenIdRefill(tokenTypeId)
	return key
}

func CreateHistoryBalanceKey(address types.Address, tokenTypeId types.TokenTypeId, snapshotHeight uint64) BalanceHistoryKey {
	key := BalanceHistoryKey{}
	key[0] = BalanceHistoryKeyPrefix
	key.AddressRefill(address)
	key.TokenIdRefill(tokenTypeId)
	key.HeightRefill(snapshotHeight)
	return key
}

func CreateCodeKey(address types.Address) CodeKey {
	key := CodeKey{}
	key[0] = CodeKeyPrefix
	key.AddressRefill(address)
	return key
}

func CreateContractMetaKey(address types.Address) ContractMetaKey {
	key := ContractMetaKey{}
	key[0] = ContractMetaKeyPrefix
	key.AddressRefill(address)
	return key
}

func CreateGidContractKey(gid types.Gid, address *types.Address) GidContractKey {
	key := GidContractKey{}
	key[0] = GidContractKeyPrefix
	key.GidRefill(gid)
	key.AddressRefill(*address)
	return key
}

func CreateGidContractPrefixKey(gid *types.Gid) []byte {
	key := make([]byte, 0, 1+types.GidSize)

	key = append(key, GidContractKeyPrefix)
	key = append(key, gid.Bytes()...)

	return key
}

func CreateVmLogListKey(logHash *types.Hash) VmLogListKey {
	key := VmLogListKey{}
	key[0] = VmLogListKeyPrefix
	key.HashRefill(*logHash)
	return key
}

func CreateCallDepthKey(blockHash types.Hash) CallDepthKey {
	key := CallDepthKey{}
	key[0] = CallDepthKeyPrefix
	key.HashRefill(blockHash)
	return key
}

// ====== state redo ======

func CreateRedoSnapshot(snapshotHeight uint64) SnapshotKey {
	key := SnapshotKey{}
	key[0] = SnapshotKeyPrefix
	key.HeightRefill(snapshotHeight)
	return key
}
