package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *Cache) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// insert snapshot block
	//cache.setLatestSnapshotBlock(snapshotBlock)
	cache.ds.InsertSnapshotBlock(snapshotBlock)

	// set latest block
	cache.hd.SetLatestSnapshotBlock(snapshotBlock)

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
	if err := cache.quotaList.Rollback(deletedChunks); err != nil {
		return err
	}

	// delete all confirmed block
	cache.unconfirmedPool.DeleteAllBlocks()

	// delete data set
	for _, chunk := range deletedChunks {
		if chunk.SnapshotBlock != nil {
			cache.ds.DeleteSnapshotBlock(chunk.SnapshotBlock)
		}

		// delete all block
		cache.hd.DeleteAccountBlocks(chunk.AccountBlocks)

		cache.ds.DeleteAccountBlocks(chunk.AccountBlocks)
	}

	// recover unconfirmed pool
	for _, block := range unconfirmedBlocks {
		cache.insertAccountBlock(block)
	}

	return nil
}

func (cache *Cache) IsSnapshotBlockExisted(hash types.Hash) bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.ds.IsSnapshotBlockExisted(hash)
}

func (cache *Cache) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return cache.hd.GetLatestSnapshotBlock()
}

func (cache *Cache) GetSnapshotHeaderByHash(hash types.Hash) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.ds.GetSnapshotBlock(hash)
}

func (cache *Cache) GetSnapshotBlockByHash(hash types.Hash) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.ds.GetSnapshotBlock(hash)
}

func (cache *Cache) GetSnapshotHeaderByHeight(height uint64) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.ds.GetSnapshotBlockByHeight(height)
}

func (cache *Cache) GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.ds.GetSnapshotBlockByHeight(height)
}
