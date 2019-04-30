package chain_db

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (store *Store) WriteDirectly(batch *leveldb.Batch) {
	store.putMemDb(batch)

	batch.Replay(store.snapshotBatch)
}

func (store *Store) WriteAccountBlock(batch *leveldb.Batch, block *ledger.AccountBlock) {
	store.WriteAccountBlockByHash(batch, block.Hash)
}

func (store *Store) WriteAccountBlockByHash(batch *leveldb.Batch, blockHash types.Hash) {

	// write store.memDb
	store.putMemDb(batch)

	store.unconfirmedBatchsLock.Lock()
	defer store.unconfirmedBatchsLock.Unlock()

	// write store.unconfirmedBatch
	store.unconfirmedBatchs.Put(blockHash, batch)
}

// snapshot
func (store *Store) WriteSnapshot(snapshotBatch *leveldb.Batch, accountBlocks []*ledger.AccountBlock) {

	accountBlockHashList := make([]types.Hash, len(accountBlocks))

	for index, accountBlock := range accountBlocks {
		accountBlockHashList[index] = accountBlock.Hash
	}

	store.WriteSnapshotByHash(snapshotBatch, accountBlockHashList)
}

// snapshot
func (store *Store) WriteSnapshotByHash(snapshotBatch *leveldb.Batch, blockHashList []types.Hash) {
	// write store.memDb
	if snapshotBatch != nil {
		store.putMemDb(snapshotBatch)
	}

	store.unconfirmedBatchsLock.Lock()
	fmt.Println(store.unconfirmedBatchs.Size())
	for _, blockHash := range blockHashList {
		batch, ok := store.unconfirmedBatchs.Get(blockHash)

		if !ok {
			panic(fmt.Sprintf("store.WriteSnapshot failed, account block hash %s batch is not existed", blockHash))
		}
		// patch
		batch.(*leveldb.Batch).Replay(store.snapshotBatch)

		// remove
		store.unconfirmedBatchs.Remove(blockHash)
	}
	fmt.Println(store.unconfirmedBatchs.Size())
	store.unconfirmedBatchsLock.Unlock()

	// write store snapshot batch
	if snapshotBatch != nil {
		snapshotBatch.Replay(store.snapshotBatch)
	}
}
