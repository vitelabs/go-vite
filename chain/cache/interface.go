package chain_cache

import (
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)

	GetSubLedger(endHeight, startHeight uint64) ([]*ledger.SnapshotChunk, error)
	GetSubLedgerAfterHeight(height uint64) ([]*ledger.SnapshotChunk, error)
}
