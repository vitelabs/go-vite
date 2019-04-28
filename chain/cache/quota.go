package chain_cache

import "github.com/vitelabs/go-vite/common/types"

func (cache *Cache) GetQuotaUsed(addr *types.Address) (uint64, uint64) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.quotaList.GetQuotaUsed(addr)
}

func (cache *Cache) GetSnapshotQuotaUsed(addr *types.Address) (uint64, uint64) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.quotaList.GetSnapshotQuotaUsed(addr)
}
