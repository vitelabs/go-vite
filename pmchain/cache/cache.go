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
	quotaList       *quotaList
}

func NewCache(chain Chain) (*Cache, error) {
	ds := NewDataSet()
	c := &Cache{
		ds:              ds,
		chain:           chain,
		unconfirmedPool: NewUnconfirmedPool(ds),
		hd:              newHotData(ds),
		quotaList:       newQuotaList(chain),
	}

	return c, nil
}

// ====== Account blocks ======
func (cache *Cache) IsAccountBlockExisted(hash *types.Hash) bool {
	return cache.ds.IsDataExisted(hash)
}

// TODO
func (cache *Cache) GetLatestAccountBlock(address *types.Address) *ledger.AccountBlock {
	return nil
}

func (cache *Cache) InsertAccountBlock(block *ledger.AccountBlock) {
	cache.quotaList.Add(&block.AccountAddress, block.Quota)

	dataId := cache.ds.InsertAccountBlock(block)
	cache.unconfirmedPool.InsertAccountBlock(&block.AccountAddress, dataId)
}

// ====== Unconfirmed blocks ======
func (cache *Cache) DeleteUnconfirmedAccountBlocks(blocks []*ledger.AccountBlock) {
	cache.unconfirmedPool.DeleteBlocks(blocks)
	for _, block := range blocks {
		cache.quotaList.Sub(&block.AccountAddress, block.Quota)
	}
}

func (cache *Cache) GetUnconfirmedBlocks() []*ledger.AccountBlock {
	return cache.unconfirmedPool.GetBlocks()
}

func (cache *Cache) GetUnconfirmedBlocksByAddress(address *types.Address) []*ledger.AccountBlock {
	return cache.unconfirmedPool.GetBlocksByAddress(address)
}

func (cache *Cache) CleanUnconfirmedPool() {
	cache.unconfirmedPool.DeleteAllBlocks()
}

// ====== Snapshot block ======
func (cache *Cache) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) {
	cache.setLatestSnapshotBlock(snapshotBlock)
}

func (cache *Cache) IsSnapshotBlockExisted(hash *types.Hash) bool {
	return cache.ds.IsDataExisted(hash)
}

func (cache *Cache) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return cache.hd.GetLatestSnapshotBlock()
}

func (cache *Cache) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return cache.hd.GetGenesisSnapshotBlock()
}

// ====== Quota ======
func (cache *Cache) GetQuotaUsed(addr *types.Address) (uint64, uint64) {
	return cache.quotaList.GetQuotaUsed(addr)
}

// ====== Delete ======
func (cache *Cache) Delete(sbList []*ledger.SnapshotBlock, subLedger map[types.Address][]*ledger.AccountBlock) error {
	cache.quotaList.Rollback(len(sbList))
	return cache.initLatestSnapshotBlock()
}

func (cache *Cache) setLatestSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) uint64 {
	dataId := cache.ds.InsertSnapshotBlock(snapshotBlock)
	cache.hd.UpdateLatestSnapshotBlock(dataId)
	return dataId
}

func (cache *Cache) Destroy() {
	cache.ds = nil
	cache.unconfirmedPool = nil
	cache.hd = nil
}
