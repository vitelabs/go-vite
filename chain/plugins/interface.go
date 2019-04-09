package chain_plugins

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	NewDb(dirName string) (*leveldb.DB, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetSnapshotBlocksByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)

	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)

	GetAccount(address *types.Address) (*ledger.Account, error)
	IsAccountBlockExisted(hash types.Hash) (bool, error)
	IsGenesisAccountBlock(block *ledger.AccountBlock) bool
}

type hashChunk struct {
	SnapshotHashHeight ledger.HashHeight

	AccountHashList []types.Hash
}

type Plugin interface {
	InsertAccountBlocks([]*ledger.AccountBlock) error
	InsertSnapshotBlocks([]*ledger.SnapshotBlock) error

	DeleteChunks([]*ledger.SnapshotChunk) error

	DeleteChunksByHash([]hashChunk) error
}
