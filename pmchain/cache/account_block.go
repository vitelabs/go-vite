package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *Cache) GetAccountBlockByHeight(addr *types.Address, height uint64) *ledger.AccountBlock {
	if dataId := cache.indexes.GetAccountBlockByHeight(addr, height); dataId > 0 {
		return cache.ds.GetAccountBlock(dataId)
	}
	return nil
}

func (cache *Cache) GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock {
	if dataId := cache.indexes.GetAccountBlockByHash(hash); dataId > 0 {
		return cache.ds.GetAccountBlock(dataId)
	}
	return nil
}
