package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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

// ====== Account blocks ======
func (cache *Cache) IsAccountBlockExisted(hash *types.Hash) bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.IsDataExisted(hash)
}

// TODO
func (cache *Cache) GetLatestAccountBlock(address *types.Address) *ledger.AccountBlock {
	return nil
}

func (cache *Cache) InsertAccountBlock(block *ledger.AccountBlock) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.quotaList.Add(&block.AccountAddress, block.Quota)
	dataId := cache.ds.InsertAccountBlock(block)
	cache.unconfirmedPool.InsertAccountBlock(&block.AccountAddress, dataId)

}

// ====== Unconfirmed blocks ======
func (cache *Cache) SnapshotAccountBlocks(blocks []*ledger.AccountBlock) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.unconfirmedPool.DeleteBlocks(blocks)
}

func (cache *Cache) GetUnconfirmedBlocks() []*ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.unconfirmedPool.GetBlocks()
}

func (cache *Cache) GetUnconfirmedBlocksByAddress(address *types.Address) []*ledger.AccountBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.unconfirmedPool.GetBlocksByAddress(address)
}

func (cache *Cache) CleanUnconfirmedPool() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.unconfirmedPool.DeleteAllBlocks()
}

// ====== Snapshot block ======
func (cache *Cache) InsertSnapshotBlock(
	snapshotBlock *ledger.SnapshotBlock,
	confirmedBlocks []*ledger.AccountBlock,
	invalidBlocks []*ledger.AccountBlock) {

	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.setLatestSnapshotBlock(snapshotBlock)

	cache.unconfirmedPool.DeleteBlocks(confirmedBlocks)
	for _, block := range invalidBlocks {
		cache.quotaList.Sub(&block.AccountAddress, block.Quota)
	}
	cache.unconfirmedPool.DeleteBlocks(invalidBlocks)

	cache.quotaList.NewNext()
}

func (cache *Cache) IsSnapshotBlockExisted(hash *types.Hash) bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.IsDataExisted(hash)
}

func (cache *Cache) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.hd.GetLatestSnapshotBlock()
}

func (cache *Cache) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.hd.GetGenesisSnapshotBlock()
}

// ====== Quota ======
func (cache *Cache) GetQuotaUsed(addr *types.Address) (uint64, uint64) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.quotaList.GetQuotaUsed(addr)
}

// ====== Delete ======
func (cache *Cache) Delete(sbList []*ledger.SnapshotBlock, subLedger map[types.Address][]*ledger.AccountBlock) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.quotaList.Rollback(len(sbList))
	return cache.initLatestSnapshotBlock()
}

func (cache *Cache) setLatestSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) uint64 {
	dataId := cache.ds.InsertSnapshotBlock(snapshotBlock)
	cache.hd.UpdateLatestSnapshotBlock(dataId)
	return dataId
}

func (cache *Cache) Destroy() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.ds = nil
	cache.unconfirmedPool = nil
	cache.hd = nil
}
