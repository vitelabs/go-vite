package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	GetSnapshotHeightByHash(hash types.Hash) (uint64, error)

	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock

	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
}
