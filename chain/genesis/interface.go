package chain_genesis

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

type Chain interface {
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) (invalidAccountBlocks []*ledger.AccountBlock, err error)
	InsertAccountBlock(vmAccountBlocks *vm_db.VmAccountBlock) error
	QuerySnapshotBlockByHeight(uint64) (*ledger.SnapshotBlock, error)
	GetContentNeedSnapshot() ledger.SnapshotContent
}
