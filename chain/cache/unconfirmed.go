package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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

func (cache *Cache) recoverUnconfirmedPool(accountBlocks []*ledger.AccountBlock) {

	for _, accountBlock := range accountBlocks {
		dataId := cache.ds.InsertAccountBlock(accountBlock)
		cache.unconfirmedPool.InsertAccountBlock(&accountBlock.AccountAddress, dataId)
	}
}
