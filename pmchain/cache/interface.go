package chain_cache

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
)

type Chain interface {
	QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	GetSubLedger(endHeight, startHeight uint64) ([]*chain_block.SnapshotSegment, error)

	GetSubLedgerAfterHeight(height uint64) ([]*chain_block.SnapshotSegment, error)
}
