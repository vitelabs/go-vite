package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Cache struct {
	chain Chain
	ds    *dataSet

	unconfirmedPool *UnconfirmedPool
	indexes         *indexes
	hd              *hotData
}

func NewCache(chain Chain) (*Cache, error) {
	ds := NewDataSet()
	c := &Cache{
		ds:              ds,
		chain:           chain,
		unconfirmedPool: NewUnconfirmedPool(ds),
		indexes:         NewIndexes(ds),
		hd:              newHotData(ds),
	}
	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

func (cache *Cache) InsertUnconfirmedAccountBlock(block *ledger.AccountBlock) {
	dataId := cache.ds.InsertAccountBlock(block)

	cache.indexes.InsertAccountBlock(dataId)
	cache.unconfirmedPool.InsertAccountBlock(dataId)
}

func (cache *Cache) GetCurrentUnconfirmedBlocks() []*ledger.AccountBlock {
	return cache.unconfirmedPool.GetCurrentBlocks()
}

func (cache *Cache) DeleteUnconfirmedSubLedger(subLedger map[types.Address][]*ledger.AccountBlock) {
	// cache.indexes.DeleteAccountBlocks(blocks)
	// cache.unconfirmedPool.InsertAccountBlock(block)
}

func (cache *Cache) UpdateLatestSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) {
	dataId := cache.ds.InsertSnapshotBlock(snapshotBlock)
	cache.hd.UpdateLatestSnapshotBlock(dataId)

}

func (cache *Cache) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return cache.hd.GetLatestSnapshotBlock()
}

func (cache *Cache) CleanUnconfirmedPool() {

}

func (cache *Cache) Destroy() {}
