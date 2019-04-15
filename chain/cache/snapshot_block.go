package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *Cache) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) {

	cache.mu.Lock()
	defer cache.mu.Unlock()

	// set latest block
	cache.setLatestSnapshotBlock(snapshotBlock)

	// delete confirmed blocks
	cache.unconfirmedPool.DeleteBlocks(confirmedBlocks)

	// new quota
	cache.quotaList.NewNext(confirmedBlocks)
}

func (cache *Cache) RollbackSnapshotBlocks(deletedChunks []*ledger.SnapshotChunk, unconfirmedBlocks []*ledger.AccountBlock) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// update latest snapshot block
	if err := cache.initLatestSnapshotBlock(); err != nil {
		cache.mu.Unlock()
		return err
	}

	// rollback quota list
	deletedChunksLength := len(deletedChunks)
	rollbackNumber := deletedChunksLength - 1

	if deletedChunks[deletedChunksLength-1].SnapshotBlock != nil {
		rollbackNumber = deletedChunksLength
	}

	if err := cache.quotaList.Rollback(rollbackNumber); err != nil {
		return err
	}

	// delete all confirmed block
	cache.unconfirmedPool.DeleteAllBlocks()
	// recover unconfirmed pool
	cache.recoverUnconfirmedPool(unconfirmedBlocks)
	return nil
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

func (cache *Cache) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.hd.GetLatestSnapshotBlock()
}

func (cache *Cache) GetSnapshotHeaderByHash(hash *types.Hash) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return nil
}

func (cache *Cache) GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return nil
}

func (cache *Cache) GetSnapshotHeaderByHeight(height uint64) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return nil
}

func (cache *Cache) GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return nil
}

func (cache *Cache) setLatestSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) uint64 {
	dataId := cache.ds.InsertSnapshotBlock(snapshotBlock)
	cache.hd.UpdateLatestSnapshotBlock(dataId)
	return dataId
}
