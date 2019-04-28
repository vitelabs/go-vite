package chain_db

import (
	"github.com/vitelabs/go-vite/common/db"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/types"
	"sync"
)

var batchPool = sync.Pool{
	New: func() interface{} {
		return new(leveldb.Batch)
	},
}

func (store *Store) Id() types.Hash {
	return store.id
}

// assume lock write when prepare
func (store *Store) Prepare() {
	if store.flushingBatch != nil {
		panic("prepare repeatedly")
	}

	store.flushingBatch = store.snapshotBatch
	store.snapshotBatch = store.getNewBatch()
}

// assume lock write when cancel prepare
func (store *Store) CancelPrepare() {
	currentSnapshotBatch := store.snapshotBatch

	store.snapshotBatch = store.getNewBatch()

	store.flushingBatch.Replay(store.snapshotBatch)
	currentSnapshotBatch.Replay(store.snapshotBatch)

	store.releaseFlushingBatch()
}

func (store *Store) RedoLog() ([]byte, error) {
	return store.flushingBatch.Dump(), nil
}

func (store *Store) Commit() error {
	if err := store.db.Write(store.flushingBatch, nil); err != nil {
		return err
	}
	return nil
}

func (store *Store) PatchRedoLog(redoLog []byte) error {
	batch := new(leveldb.Batch)

	if err := batch.Load(redoLog); err != nil {
		return err
	}

	if err := store.db.Write(batch, nil); err != nil {
		return err
	}

	return nil
}

// assume lock write when call after commit
func (store *Store) AfterCommit() {
	// reset flushing batch
	store.releaseFlushingBatch()

	// reset mem db
	store.memDbMu.Lock()

	store.memDb = db.NewMemDB()

	// replay snapshot batch
	store.snapshotBatch.Replay(store.memDb)

	// replay unconfirmed batch
	iter := store.unconfirmedBatchs.Iterator()
	for iter.Next() {
		iter.Value().(*leveldb.Batch).Replay(store.memDb)
	}

	store.memDbMu.Unlock()
}

func (store *Store) getNewBatch() *leveldb.Batch {
	batch := batchPool.Get().(*leveldb.Batch)
	batch.Reset()
	return batch
}

func (store *Store) releaseFlushingBatch() {
	batchPool.Put(store.flushingBatch)
	store.flushingBatch = nil
}
