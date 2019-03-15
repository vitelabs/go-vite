package chain_cache

import "github.com/vitelabs/go-vite/ledger"

type Chain interface {
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
}
