package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
)

type DB interface {
	NewBatch() Batch
	Write(Batch) error
}

type Batch interface {
	Put(key, value []byte)
}

type MemDB interface {
	Put(blockHash *types.Hash, key, value []byte)
	Get(key []byte) ([]byte, bool)

	GetAndDelete(blockHash *types.Hash) ([][]byte, [][]byte)
}
