package chain_genesis

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) (invalidSubLedger map[types.Address][]*ledger.AccountBlock, err error)
	GetLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetContentNeedSnapshot() ledger.SnapshotContent
}
