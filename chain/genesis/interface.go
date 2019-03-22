package chain_genesis

import (
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) (invalidAccountBlocks []*ledger.AccountBlock, err error)
	GetSnapshotHeaderByHeight(uint64) (*ledger.SnapshotBlock, error)
	GetContentNeedSnapshot() ledger.SnapshotContent
}
