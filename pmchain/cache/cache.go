package chain_cache

import (
	"github.com/vitelabs/go-vite/ledger"
)

type Cache struct {
	ds *dataSet

	unconfirmedPool *UnconfirmedPool
	indexes         *indexes
}

func NewCache() *Cache {
	ds := NewDataSet()
	return &Cache{
		ds: ds,

		unconfirmedPool: NewUnconfirmedPool(ds),
		indexes:         NewIndexes(ds),
	}
}

func (cache *Cache) UnconfirmedPool() *UnconfirmedPool {
	return cache.unconfirmedPool
}

func (cache *Cache) InsertUnconfirmedAccountBlock(block *ledger.AccountBlock) {
	dataId := cache.ds.InsertAccountBlock(block)

	cache.indexes.InsertAccountBlock(dataId)
	cache.unconfirmedPool.InsertAccountBlock(dataId)
}

func (cache *Cache) GetCurrentUnconfirmedBlocks() []*ledger.AccountBlock {
	return cache.unconfirmedPool.GetCurrentBlocks()
}

// TODO
//func (cache *Cache) DeleteUnconfirmedAccountBlock(blocks []*ledger.AccountBlock) {
//	cache.indexes.DeleteAccountBlocks(blocks)
//	//cache.unconfirmedPool.InsertAccountBlock(block)
//}

func (cache *Cache) Destroy() {}
