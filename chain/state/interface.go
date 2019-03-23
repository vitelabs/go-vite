package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	IsAccountBlockExisted(hash *types.Hash) (bool, error)

	GetAccountId(address *types.Address) (uint64, error)

	GetConfirmedTimes(blockHash *types.Hash) (uint64, error)

	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	GetAccountBlocks(blockHash *types.Hash, count uint64) ([]*ledger.AccountBlock, error)
}
