package chain_cache

import (
	"github.com/vitelabs/go-vite/interfaces"
	"sync"
)

type Cache struct {
	chain Chain

	ds *dataSet
	mu sync.RWMutex

	unconfirmedPool *UnconfirmedPool
	hd              *hotData
	quotaList       *quotaList
}

func NewCache(chain Chain) (*Cache, error) {
	ds := NewDataSet()
	c := &Cache{
		ds:    ds,
		chain: chain,

		unconfirmedPool: NewUnconfirmedPool(ds),
		hd:              newHotData(ds),
		quotaList:       newQuotaList(chain),
	}

	return c, nil
}

func (cache *Cache) Destroy() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.ds.Close()
	cache.ds = nil
	cache.unconfirmedPool = nil
	cache.hd = nil
}

func (cache *Cache) GetStatus() []interfaces.DBStatus {
	dsCacheStatus := cache.ds.GetStatus()

	return []interfaces.DBStatus{{
		Name:   "blockCache.ds.cache",
		Count:  dsCacheStatus[0].Count,
		Size:   dsCacheStatus[0].Size,
		Status: "",
	}}
}
