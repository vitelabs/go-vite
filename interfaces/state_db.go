package interfaces

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type StorageIterator interface {
	Prev() bool
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
}
