package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *Cache) InsertAccountBlock(block *ledger.AccountBlock) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.quotaList.Add(block.AccountAddress, block.Quota)
	dataId := cache.ds.InsertAccountBlock(block)
	cache.unconfirmedPool.InsertAccountBlock(&block.AccountAddress, dataId)
}

func (cache *Cache) RollbackAccountBlocks(accountBlocks []*ledger.AccountBlock) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// delete all confirmed block
	cache.unconfirmedPool.DeleteBlocks(accountBlocks)

	// rollback quota list
	for _, block := range accountBlocks {
		cache.quotaList.Sub(block.AccountAddress, block.Quota)
	}

	return nil
}

func (cache *Cache) IsAccountBlockExisted(hash *types.Hash) bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.IsDataExisted(hash)
}

// TODO
func (cache *Cache) GetLatestAccountBlock(address types.Address) *ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.GetLatestAccountBlock(address)
}

func (cache *Cache) GetAccountBlockByHeight(addr types.Address, height uint64) *ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.GetAccountBlockByHeight(addr, height)
}

func (cache *Cache) GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.GetAccountBlockByHash(hash)
}
