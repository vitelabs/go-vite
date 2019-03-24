package chain_cache

import (
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	GetSubLedger(endHeight, startHeight uint64) ([]*chain_block.SnapshotSegment, error)

	GetSubLedgerAfterHeight(height uint64) ([]*chain_block.SnapshotSegment, error)
}
