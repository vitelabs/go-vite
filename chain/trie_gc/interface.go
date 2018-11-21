package trie_gc

import (
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/ledger"
)

type Collector interface {
}

type Chain interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward bool, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
	ChainDb() *chain_db.ChainDb
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
}
