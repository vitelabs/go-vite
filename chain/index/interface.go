package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	IsGenesisAccountBlock(hash types.Hash) bool
}
type Store interface {
	NewBatch() interfaces.Batch
	NewIterator(slice *util.Range) iterator.Iterator
	Write(interfaces.Batch) error
	Has(key []byte) (bool, error)
	Get(key []byte) ([]byte, error)
	Clean() error

	Close() error
}
