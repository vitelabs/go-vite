package chain_plugins

import "github.com/vitelabs/go-vite/common/types"

const (
	OnRoadInfoKeyPrefix = byte(1)

	DiffTokenHash = byte(2)
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
