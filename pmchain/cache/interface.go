package chain_cache

import "github.com/vitelabs/go-vite/ledger"

type Chain interface {
	GetLatestSnapshotBlock() (*ledger.SnapshotBlock, error)

	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
}
