package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *Cache) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.hd.GetLatestSnapshotBlock()
}

func (cache *Cache) GetSnapshotHeaderByHash(hash *types.Hash) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return nil
}

func (cache *Cache) GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return nil
}

func (cache *Cache) GetSnapshotHeaderByHeight(height uint64) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return nil
}

func (cache *Cache) GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return nil
}
