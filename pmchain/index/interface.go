package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type Store interface {
	NewBatch() Batch
	NewIterator(slice *util.Range) iterator.Iterator
	Write(Batch) error
	Has(key []byte) (bool, error)
	Get(key []byte) ([]byte, error)
	Clean() error

	Close() error
}

type Batch interface {
	Put(key, value []byte)
	Delete(key []byte)
}
