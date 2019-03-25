package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *Cache) GetSnapshotHeaderByHash(hash *types.Hash) *ledger.SnapshotBlock {
	return nil
}

func (cache *Cache) GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock {
	return nil
}

func (cache *Cache) GetSnapshotHeaderByHeight(height uint64) *ledger.SnapshotBlock {
	return nil
}

func (cache *Cache) GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock {
	return nil
}
