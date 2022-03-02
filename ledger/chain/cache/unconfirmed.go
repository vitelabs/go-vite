package chain_cache

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

func (cache *Cache) GetUnconfirmedBlocks() []*ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.unconfirmedPool.GetBlocks()
}

func (cache *Cache) GetUnconfirmedBlocksByAddress(address *types.Address) []*ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.unconfirmedPool.GetBlocksByAddress(address)
}
