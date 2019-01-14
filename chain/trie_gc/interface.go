package trie_gc

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type Collector interface {
	Start()
	Stop()
	Status() uint8
}

type Chain interface {
	vm_context.Chain

	AccountType(address *types.Address) (uint64, error)
	GetLatestBlockEventId() (uint64, error)
	GetEvent(eventId uint64) (byte, []types.Hash, error)
	ChainDb() *chain_db.ChainDb

	TrieDb() *leveldb.DB
	CleanTrieNodePool()
	GenStateTrieFromDb(prevStateHash types.Hash, snapshotContent ledger.SnapshotContent) (*trie.Trie, error)

	GetSecondSnapshotBlock() *ledger.SnapshotBlock

	GetGenesisMintageBlockVC() vmctxt_interface.VmDatabase
	GetGenesisMintageSendBlockVC() vmctxt_interface.VmDatabase
	GetGenesisConsensusGroupBlockVC() vmctxt_interface.VmDatabase
	GetGenesisRegisterBlockVC() vmctxt_interface.VmDatabase

	IsGenesisSnapshotBlock(block *ledger.SnapshotBlock) bool
	IsGenesisAccountBlock(block *ledger.AccountBlock) bool

	StopSaveTrie()
	StartSaveTrie()
}
