package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
)

type Store interface {
	NewBatch() Batch
	Write(Batch) error
	Has(key []byte) (bool, error)
	Clean() error

	Close() error
}

type Batch interface {
	Put(key, value []byte)
	Delete(key []byte)
}

type MemDB interface {
	Put(blockHash *types.Hash, key, value []byte)
	Get(key []byte) ([]byte, bool)
	Has(key []byte) bool

	GetByBlockHash(blockHash *types.Hash) ([][]byte, [][]byte)

	DeleteByBlockHash(blockHash *types.Hash)

	Clean()
}
