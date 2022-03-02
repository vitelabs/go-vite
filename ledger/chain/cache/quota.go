package chain_cache

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
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
