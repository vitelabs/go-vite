package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
)

type NeedSnapshotCache struct {
	chain           *Chain
	snapshotContent ledger.SnapshotContent

	lock sync.Mutex
	log  log15.Logger
}

func NewNeedSnapshotContent(chain *Chain) *NeedSnapshotCache {
	return &NeedSnapshotCache{
		chain:           chain,
		snapshotContent: ledger.SnapshotContent{},
		log:             log15.New("module", "chain/NeedSnapshotCache"),
	}
}

func (cache *NeedSnapshotCache) Add(addr *types.Address, accountBlockHeight uint64, accountBlockHash *types.Hash) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if cachedSnapshotContentItem := cache.snapshotContent[*addr]; cachedSnapshotContentItem != nil && cachedSnapshotContentItem.AccountBlockHeight >= accountBlockHeight {
		cache.log.Error("Can't add", "method", "Add")
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
		cache.log.Error("Can't remove", "method", "Remove")
		return
	}

	if cachedSnapshotContentItem.AccountBlockHeight <= height {
		delete(cache.snapshotContent, *addr)
	}
}
