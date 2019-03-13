package chain_state

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
}

type StateSnapshot interface {
	// ====== balance ======
	GetBalance(tokenId *types.TokenTypeId) (*big.Int, error)

	// ====== code ======
	GetCode() ([]byte, error)

	// ====== Storage ======
	GetValue([]byte) ([]byte, error)

	NewStorageIterator(prefix []byte) StorageIterator
}
