package chain_genesis

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	ledger "github.com/vitelabs/go-vite/interfaces/core"
)

type Chain interface {
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) (invalidAccountBlocks []*ledger.AccountBlock, err error)
	InsertAccountBlock(vmAccountBlocks *interfaces.VmAccountBlock) error
	QuerySnapshotBlockByHeight(uint64) (*ledger.SnapshotBlock, error)
	GetContentNeedSnapshot() ledger.SnapshotContent

	WriteGenesisCheckSum(hash types.Hash) error
	QueryGenesisCheckSum() (*types.Hash, error)
}
