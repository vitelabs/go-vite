package trie_gc

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
)

type Collector interface {
	Start()
	Stop()
	Status() uint8
}

type Chain interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward bool, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
	GetLatestBlockEventId() (uint64, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetEvent(eventId uint64) (byte, []types.Hash, error)
	ChainDb() *chain_db.ChainDb
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	TrieDb() *leveldb.DB
	CleanTrieNodePool()
	GenStateTrieFromDb(prevStateHash types.Hash, snapshotContent ledger.SnapshotContent) (*trie.Trie, error)

	StopSaveTrie()
	StartSaveTrie()
}
