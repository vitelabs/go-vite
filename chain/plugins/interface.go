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
