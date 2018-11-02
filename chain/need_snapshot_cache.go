package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
)

type NeedSnapshotCache struct {
	chain    *chain
	cacheMap map[types.Address]*ledger.AccountBlock

	lock sync.Mutex
	log  log15.Logger
}

func NewNeedSnapshotContent(chain *chain, unconfirmedSubLedger map[types.Address][]*ledger.AccountBlock) *NeedSnapshotCache {
	cache := &NeedSnapshotCache{
		chain:    chain,
		cacheMap: make(map[types.Address]*ledger.AccountBlock),
		log:      log15.New("module", "chain/NeedSnapshotCache"),
	}

	for addr, blocks := range unconfirmedSubLedger {
		cache.cacheMap[addr] = blocks[0]
	}
	return cache
}

func (cache *NeedSnapshotCache) GetSnapshotContent() ledger.SnapshotContent {
	content := make(ledger.SnapshotContent, 0)
	for addr, block := range cache.cacheMap {
		content[addr] = &ledger.HashHeight{
			Height: block.Height,
			Hash:   block.Hash,
		}
	}
	return content
}

func (cache *NeedSnapshotCache) GetBlockByHash(addr *types.Address, hash types.Hash) *ledger.AccountBlock {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	block := cache.cacheMap[*addr]
	if block == nil {
		return nil
	}
	if block.Hash == hash {
		return block
	}

	return nil
}

func (cache *NeedSnapshotCache) Get(addr *types.Address) *ledger.AccountBlock {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	return cache.cacheMap[*addr]
}

func (cache *NeedSnapshotCache) Set(subLedger map[types.Address]*ledger.AccountBlock) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	for addr, accountBlock := range subLedger {
		if cachedItem := cache.cacheMap[addr]; cachedItem != nil && cachedItem.Height >= accountBlock.Height {
			cache.unsavePrintCacheMap()
			cache.log.Crit("cachedItem.Height > accountBlock.Height", "method", "Set")
			return
		}

		cache.cacheMap[addr] = accountBlock
	}

}

func (cache *NeedSnapshotCache) unsavePrintCacheMap() {
	for addr, block := range cache.cacheMap {
		cache.log.Error(fmt.Sprintf("%s %+v\n", addr, block))
	}
}

func (cache *NeedSnapshotCache) BeSnapshot(addr *types.Address, height uint64) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cachedItem := cache.cacheMap[*addr]
	if cachedItem == nil {
		cache.unsavePrintCacheMap()
		cache.log.Crit("cacheItem is nil", "method", "BeSnapshot")
	}

	if cachedItem.Height < height {
		cache.unsavePrintCacheMap()
		cache.log.Crit("cacheItem.Height < height", "method", "BeSnapshot")
	}

	if cachedItem.Height == height {
		delete(cache.cacheMap, *addr)
	}
}

func (cache *NeedSnapshotCache) Remove(addrList []types.Address) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	for _, addr := range addrList {
		delete(cache.cacheMap, addr)
	}
}
