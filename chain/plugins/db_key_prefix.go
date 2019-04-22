package chain_plugins

import "github.com/vitelabs/go-vite/common/types"

const (
	OnRoadInfoKeyPrefix = byte(1)

	DiffTokenHash = byte(2)

	onRoadPrefixKey = byte(3)

	OnRoadKey = byte(4)
)

func CreateOnRoadInfoKey(addr *types.Address, tId *types.TokenTypeId) []byte {
	key := make([]byte, 0, 1+types.AddressSize+types.TokenTypeIdSize)
	key = append(key, OnRoadInfoKeyPrefix)
	key = append(key, addr.Bytes()...)
	key = append(key, tId.Bytes()...)
	return key
}

func CreateOnRoadInfoPrefixKey(addr *types.Address) []byte {
	key := make([]byte, 0, 1+types.AddressSize)
	key = append(key, OnRoadInfoKeyPrefix)
	key = append(key, addr.Bytes()...)
	return key
}

func CreateOnRoadPrefixKey(addr *types.Address) []byte {
	key := make([]byte, 0, 1+types.AddressSize)
	key = append(key, onRoadPrefixKey)
	key = append(key, addr.Bytes()...)
	return key
}

func CreateOnRoadKey(addr *types.Address, hash *types.Hash) []byte {
	key := make([]byte, 0, 1+types.AddressSize)
	key = append(key, OnRoadKey)
	key = append(key, addr.Bytes()...)
	key = append(key, hash.Bytes()...)
	return key
}
