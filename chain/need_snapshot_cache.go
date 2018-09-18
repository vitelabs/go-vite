package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
)

type NeedSnapshotCache struct {
	chain           *Chain
	snapshotContent ledger.SnapshotContent

	lock sync.Mutex
}

func NewNeedSnapshotContent(chain *Chain) *NeedSnapshotCache {
	return &NeedSnapshotCache{
		chain:           chain,
		snapshotContent: ledger.SnapshotContent{},
	}
}

func (cache *NeedSnapshotCache) Add(addr *types.Address, accountBlockHeight uint64, accountBlockHash *types.Hash) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if cachedSnapshotContentItem := cache.snapshotContent[*addr]; cachedSnapshotContentItem != nil && cachedSnapshotContentItem.AccountBlockHeight >= accountBlockHeight {
		return
	}

	snapshotContentItem := &ledger.SnapshotContentItem{
		AccountBlockHeight: accountBlockHeight,
		AccountBlockHash:   *accountBlockHash,
	}

	cache.snapshotContent[*addr] = snapshotContentItem
}

func (cache *NeedSnapshotCache) Remove(addr *types.Address, height uint64) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cachedSnapshotContentItem := cache.snapshotContent[*addr]
	if cachedSnapshotContentItem == nil {
		return
	}

	if cachedSnapshotContentItem.AccountBlockHeight <= height {
		delete(cache.snapshotContent, *addr)
	}
}
