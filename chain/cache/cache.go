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

func (cache *Cache) RollbackAccountBlocks(accountBlocks []*ledger.AccountBlock) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// delete all confirmed block
	cache.unconfirmedPool.DeleteBlocks(accountBlocks)

	// rollback quota list
	for _, block := range accountBlocks {
		cache.quotaList.Sub(&block.AccountAddress, block.Quota)
	}

	return nil
}
func (cache *Cache) RollbackSnapshotBlocks(deletedSnapshotSegments []*ledger.SnapshotChunk) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if len(deletedSnapshotSegments) <= 0 {
		return nil
	}

	// delete all confirmed block
	cache.unconfirmedPool.DeleteAllBlocks()

	// update latest snapshot block
	if err := cache.initLatestSnapshotBlock(); err != nil {
		return err
	}

	// rollback quota list
	if err := cache.quotaList.Rollback(len(deletedSnapshotSegments)); err != nil {
		return err
	}
	return nil
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

func (cache *Cache) RecoverUnconfirmedPool(accountBlocks []*ledger.AccountBlock) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for _, accountBlock := range accountBlocks {
		dataId := cache.ds.InsertAccountBlock(accountBlock)
		cache.unconfirmedPool.InsertAccountBlock(&accountBlock.AccountAddress, dataId)
	}
}

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

// ====== Snapshot block ======
func (cache *Cache) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) {

	cache.mu.Lock()
	defer cache.mu.Unlock()

	// set latest block
	cache.setLatestSnapshotBlock(snapshotBlock)

	// delete confirmed blocks
	cache.unconfirmedPool.DeleteBlocks(confirmedBlocks)

	// new quota
	cache.quotaList.NewNext()
}

func (cache *Cache) IsSnapshotBlockExisted(hash *types.Hash) bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.IsDataExisted(hash)
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

func (cache *Cache) GetSnapshotQuotaUsed(addr *types.Address) (uint64, uint64) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.quotaList.GetSnapshotQuotaUsed(addr)
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
