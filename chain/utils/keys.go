package chain_utils

import "github.com/vitelabs/go-vite/common/types"

type DBKey interface {
	Bytes() []byte
	String() string
}

type DBKeyAddressRefill interface {
	AddressRefill(addr types.Address)
}

type DBKeyHashRefill interface {
	HashRefill(hash types.Hash)
}

type DBKeyHeightRefill interface {
	HeightRefill(height uint64)
}

type DBKeyRealKeyRefill interface {
	KeyRefill(real StorageRealKey)
}

type DBKeyTokenIdRefill interface {
	TokenIdRefill(tokenId types.TokenTypeId)
}

type DBKeyStorage interface {
	DBKey
	DBKeyAddressRefill
	DBKeyRealKeyRefill
}

type DBKeyBalance interface {
	DBKey
	DBKeyAddressRefill
	DBKeyTokenIdRefill
}
