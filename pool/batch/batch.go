package batch

import (
	"errors"

	"github.com/vitelabs/go-vite/common/types"
)

var (
	// ErrorArrivedToMax means that the maximum capacity has been reached for the batch
	ErrorArrivedToMax = errors.New("arrived to max")
	// ErrorReference mean that the dependency item(account or snapshot block) does not exist in chain or batch
	ErrorReference = errors.New("refer not exist")
)

// Batch is a batch for block insertion.
type Batch interface {
	// AddAItem add account block to Batch
	AddAItem(item Item, sHash *types.Hash) error
	// AddAItem add snapshot block to Batch
	AddSItem(item Item) error
	// Levels returns all levels for the Batch
	Levels() []Level
	// Size returns the number of items
	Size() int
	// Info returns the basic info for the Batch
	Info() string
	// Version returns the version for the Batch
	Version() uint64
	// Exists returns whether or not the hash is in the Batch
	Exists(hash types.Hash) bool
	// Batch runs the Batch
	Batch(snapshotFn BucketExecutorFn, accountFn BucketExecutorFn) error
	// Id returns the id of the Batch
	Id() uint64
}

// Level is a bucket collection, and it cannot be inserted concurrently between level
type Level interface {
	// Buckets returns all buckets in the level.
	Buckets() []Bucket
	// Add will add the item to the level.
	Add(item Item) error
	// SHash is snapshot hash for the level, will return nil for snapshot level.
	SHash() *types.Hash
	// Snapshot mean whether it is a snapshot level(just snapshot item can be added).
	Snapshot() bool
	// Index means the index of the level
	Index() int
	// Close means not accepting any item
	Close()
	Closed() bool
	// Done mean all item have been added to chain
	Done()
	HasDone() bool

	// The count of Items
	Size() int
}

// Bucket is a item(account and snapshot block) collection for same address
type Bucket interface {
	// Items returns all items
	Items() []Item
	// Owner mean the same address, it will return nil if all items are snapshot item.
	Owner() *types.Address
}

// Item means a account block or a snapshot block
type Item interface {
	// keys, accounts, snapshot
	ReferHashes() ([]types.Hash, []types.Hash, *types.Hash)
	// Owner will return nil if the item is the snapshot item.
	Owner() *types.Address
	Hash() types.Hash
	Height() uint64
	PrevHash() types.Hash
}

// BucketExecutorFn can insert a bucket
type BucketExecutorFn func(p Batch, l Level, bucket Bucket, version uint64) error
