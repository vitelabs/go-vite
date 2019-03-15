package chain_index

import "github.com/vitelabs/go-vite/common/types"

func createAccountAddressKey(addr *types.Address) []byte {
	addrBytes := addr.Bytes()
	accountAddressKey := make([]byte, 0, 1+types.AddressSize)
	accountAddressKey = append(append(accountAddressKey, AccountAddressKeyPrefix), addrBytes...)
	return accountAddressKey
}

func createAccountIdKey(accountId uint64) []byte {
	accountIdKey := make([]byte, 0, 9)
	accountIdKey = append(append(accountIdKey, AccountIdKeyPrefix), Uint64ToFixedBytes(accountId)...)
	return accountIdKey
}
func createAccountBlockHashKey(blockHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(append(key, AccountBlockHashKeyPrefix), blockHash.Bytes()...)
	return key
}

func createSnapshotBlockHashKey(snapshotBlockHash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(append(append(key, SnapshotBlockHashKeyPrefix), snapshotBlockHash.Bytes()...))
	return key
}
