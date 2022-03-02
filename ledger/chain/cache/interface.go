package chain_cache

import (
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

type Chain interface {
	QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	QuerySnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)

	GetSubLedger(endHeight, startHeight uint64) ([]*ledger.SnapshotChunk, error)
	GetSubLedgerAfterHeight(height uint64) ([]*ledger.SnapshotChunk, error)
}
