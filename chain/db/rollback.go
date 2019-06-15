package chain_db

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

// rollback
func (store *Store) RollbackAccountBlocks(rollbackBatch *leveldb.Batch, accountBlocks []*ledger.AccountBlock) {
	// delete store.unconfirmedBatchMap
	for _, block := range accountBlocks {
		store.unconfirmedBatchs.Remove(block.Hash)
	}

	// write store.memDb
	store.putMemDb(rollbackBatch)
}

// rollback
func (store *Store) RollbackAccountBlockByHash(rollbackBatch *leveldb.Batch, blockHashList []types.Hash) {
	// delete store.unconfirmedBatchMap
	for _, blockHash := range blockHashList {
		// remove
		store.unconfirmedBatchs.Remove(blockHash)
	}

	// write store.memDb
	store.putMemDb(rollbackBatch)
}

func (store *Store) RollbackSnapshot(rollbackBatch *leveldb.Batch) {
	// write store.memDb
	store.putMemDb(rollbackBatch)

	// set store.snapshotBatch
	store.unconfirmedBatchs.All(func(batch *leveldb.Batch) {
		store.snapshotBatch.Append(batch)
	})

	store.snapshotBatch.Append(rollbackBatch)

	// reset
	store.unconfirmedBatchs.Clear()
}
