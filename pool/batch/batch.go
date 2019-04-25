package batch

import (
	"errors"

	"github.com/vitelabs/go-vite/common/types"
)

var (
	MAX_ERROR   = errors.New("arrived to max")
	REFER_ERROR = errors.New("refer not exist")
)

type Batch interface {
	AddItem(item Item) error
	Levels() []Level
	Size() int
	Info() string
	Version() uint64
	Exists(hash types.Hash) bool
	Id() uint64
}

type Level interface {
	Buckets() []Bucket
	Add(item Item) error
	Snapshot() bool
	Index() int
	Close()
	Closed() bool
}

type Bucket interface {
	Items() []Item
	Owner() *types.Address
}

type Item interface {
	// keys, accounts, snapshot
	ReferHashes() ([]types.Hash, []types.Hash, *types.Hash)
	Owner() *types.Address
	Hash() types.Hash
	Height() uint64
	PrevHash() types.Hash
}
