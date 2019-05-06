package chain_cache

import "github.com/vitelabs/go-vite/common/types"

func (cache *Cache) GetQuotaUsedList(addr types.Address) []types.QuotaInfo {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.quotaList.GetQuotaUsedList(addr)
}

func (cache *Cache) GetGlobalQuota() types.QuotaInfo {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.quotaList.GetGlobalQuota()
}
