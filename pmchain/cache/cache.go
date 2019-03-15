package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Cache struct {
	chain Chain

	ds *dataSet

	unconfirmedPool *UnconfirmedPool
	hd              *hotData
}

func NewCache(chain Chain) (*Cache, error) {
	ds := NewDataSet()
	c := &Cache{
		ds:              ds,
		chain:           chain,
		unconfirmedPool: NewUnconfirmedPool(ds),
		hd:              newHotData(ds),
	}

	return c, nil
}

func (cache *Cache) IsAccountBlockExisted(hash *types.Hash) bool {
	return cache.ds.IsDataExisted(hash)
}

func (cache *Cache) IsSnapshotBlockExisted(hash *types.Hash) bool {
	return cache.ds.IsDataExisted(hash)
}

func (cache *Cache) InsertUnconfirmedAccountBlock(block *ledger.AccountBlock) {
	dataId := cache.ds.InsertAccountBlock(block)

	cache.unconfirmedPool.InsertAccountBlock(dataId)
}

func (cache *Cache) GetCurrentUnconfirmedBlocks() []*ledger.AccountBlock {
	return cache.unconfirmedPool.GetCurrentBlocks()
}

func (cache *Cache) DeleteUnconfirmedBlocks(blocks []*ledger.AccountBlock) {
	cache.unconfirmedPool.DeleteBlocks(blocks)
}

func (cache *Cache) UpdateLatestSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) {
	dataId := cache.ds.InsertSnapshotBlock(snapshotBlock)
	cache.hd.UpdateLatestSnapshotBlock(dataId)
}

func (cache *Cache) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return cache.hd.GetLatestSnapshotBlock()
}

func (cache *Cache) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return cache.hd.GetGenesisSnapshotBlock()
}

func (cache *Cache) CleanUnconfirmedPool() {

}

func (cache *Cache) Destroy() {
	cache.ds = nil
	cache.unconfirmedPool = nil
	cache.hd = nil
}
