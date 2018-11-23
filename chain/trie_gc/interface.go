package trie_gc

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/ledger"
)

type Collector interface {
	Start()
	Stop()
	Status() uint8
	ClearedHeight() uint64
	MarkedHeight() uint64
}

type Chain interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward bool, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
	ChainDb() *chain_db.ChainDb
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	TrieDb() *leveldb.DB
	CleanTrieNodePool()

	StopSaveTrie()
	StartSaveTrie()
}
