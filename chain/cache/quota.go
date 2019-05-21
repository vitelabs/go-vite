package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

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

func (cache *Cache) ResetUnconfirmedQuotas(unconfirmedBlocks []*ledger.AccountBlock) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.quotaList.ResetUnconfirmedQuotas(unconfirmedBlocks)
}
