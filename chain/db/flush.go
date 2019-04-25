package chain_db

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
)

func (store *Store) Id() types.Hash {
	return store.id
}

func (store *Store) Prepare() {
	store.mu.RLock()
	defer store.mu.RUnlock()

	store.flushingBatch.Reset()
	store.snapshotMemDb.Flush(store.flushingBatch)
}

func (store *Store) CancelPrepare() {
	store.flushingBatch.Reset()
}

func (store *Store) RedoLog() ([]byte, error) {
	return store.flushingBatch.Dump(), nil
}

func (store *Store) Commit() error {
	if err := store.db.Write(store.flushingBatch, nil); err != nil {
		return err
	}
	store.flushingBatch.Reset()

	store.resetMemDB()
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

	store.resetMemDB()
	return nil
}

func (store *Store) resetMemDB() {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.snapshotMemDb.Reset()

	store.memDb = NewMemDB()

	elem := store.unconfirmedBatchList.Front()
	for elem != nil {
		elem.Value.(*leveldb.Batch).Replay(store.memDb)
		elem = elem.Next()
	}
}
