package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *Cache) InsertAccountBlock(block *ledger.AccountBlock) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.insertAccountBlock(block)
}

func (cache *Cache) RollbackAccountBlocks(accountBlocks []*ledger.AccountBlock) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// delete all confirmed block
	cache.unconfirmedPool.DeleteBlocks(accountBlocks)

	// rollback quota list
	for _, block := range accountBlocks {
		cache.quotaList.Sub(block.AccountAddress, block.Quota, block.QuotaUsed)
	}

	// delete account data
	cache.hd.DeleteAccountBlocks(accountBlocks)

	// rollback data set
	cache.ds.DeleteAccountBlocks(accountBlocks)

	return nil
}

func (cache *Cache) IsAccountBlockExisted(hash types.Hash) bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.IsAccountBlockExisted(hash)
}

func (cache *Cache) GetLatestAccountBlock(address types.Address) *ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.hd.GetLatestAccountBlock(address)
}

func (cache *Cache) GetAccountBlockByHeight(addr types.Address, height uint64) *ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.GetAccountBlockByHeight(addr, height)
}

func (cache *Cache) GetAccountBlockByHash(hash types.Hash) *ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.GetAccountBlock(hash)
}

func (cache *Cache) insertAccountBlock(block *ledger.AccountBlock) {
	// add data set
	cache.ds.InsertAccountBlock(block)

	// add hot data
	cache.hd.InsertAccountBlock(block)

	// add unconfirmed pool
	cache.unconfirmedPool.InsertAccountBlock(block)

	// add quota
	cache.quotaList.Add(block.AccountAddress, block.Quota, block.QuotaUsed)
}
