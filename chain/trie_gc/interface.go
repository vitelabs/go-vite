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
	Check() (bool, error)
	Recover() (returnErr error)
}

type Chain interface {
	vm_context.Chain

	NewGenesisSnapshotBlock() ledger.SnapshotBlock
	NewSecondSnapshotBlock() ledger.SnapshotBlock
	NewGenesisMintageBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase)
	NewGenesisMintageSendBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase)
	NewGenesisConsensusGroupBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase)
	NewGenesisRegisterBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase)

	AccountType(address *types.Address) (uint64, error)
	GetLatestBlockEventId() (uint64, error)
	GetEvent(eventId uint64) (byte, []types.Hash, error)
	ChainDb() *chain_db.ChainDb

	TrieDb() *leveldb.DB
	CleanTrieNodePool()
	GenStateTrieFromDb(prevStateHash types.Hash, snapshotContent ledger.SnapshotContent) (*trie.Trie, error)
	ShallowCheckStateTrie(stateHash *types.Hash) (bool, error)

	IsGenesisSnapshotBlock(block *ledger.SnapshotBlock) bool
	IsGenesisAccountBlock(block *ledger.AccountBlock) bool

	StopSaveTrie()
	StartSaveTrie()
}
