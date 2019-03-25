package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	IsAccountBlockExisted(hash *types.Hash) (bool, error)

	QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	GetSnapshotHeightByHash(hash *types.Hash) (uint64, error)
}
