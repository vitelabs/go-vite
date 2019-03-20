package interfaces

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type StorageIterator interface {
	Next() bool

	Key() []byte
	Value() []byte
	Error() error
	Release()
}

type StateSnapshot interface {
	// ====== balance ======
	GetBalance(tokenId *types.TokenTypeId) (*big.Int, error)

	// ====== Storage ======
	GetValue([]byte) ([]byte, error)

	NewStorageIterator(prefix []byte) StorageIterator

	Release()
}

type Batch interface {
	Put(key, value []byte)
	Delete(key []byte)
}

type Store interface {
	Get([]byte) ([]byte, error)
	Has([]byte) (bool, error)
}
