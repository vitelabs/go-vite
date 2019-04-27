package chain_plugins

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetSnapshotBlocksByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)
	GetSubLedger(startHeight, endHeight uint64) ([]*ledger.SnapshotChunk, error)
	GetSubLedgerAfterHeight(height uint64) ([]*ledger.SnapshotChunk, error)
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)

	IsAccountBlockExisted(hash types.Hash) (bool, error)
	IsGenesisAccountBlock(hash types.Hash) bool
	IsContractAccount(address types.Address) (bool, error)

	GetAllUnconfirmedBlocks() []*ledger.AccountBlock
}

type Plugin interface {
	InsertAccountBlock(*leveldb.Batch, *ledger.AccountBlock) error

	InsertSnapshotBlock(*leveldb.Batch, *ledger.SnapshotBlock, []*ledger.AccountBlock) error

	DeleteAccountBlocks(*leveldb.Batch, []*ledger.AccountBlock) error

	DeleteSnapshotBlocks(*leveldb.Batch, []*ledger.SnapshotChunk) error
}
