package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *Cache) GetAccountBlockByHeight(addr *types.Address, height uint64) *ledger.AccountBlock {
	return cache.ds.GetAccountBlockByHeight(addr, height)
}

func (cache *Cache) GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock {
	return cache.ds.GetAccountBlockByHash(hash)
}
