package chain_db

import (
	"container/list"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"os"
	"sync"
)

type Store struct {
	id            types.Hash
	mu            sync.RWMutex
	memDb         *MemDB
	snapshotMemDb *SnapshotMemDB

	unconfirmedBatchMap  map[types.Hash]*list.Element
	unconfirmedBatchList *list.List

	dbDir string
	db    *leveldb.DB

	flushingBatch *leveldb.Batch
}

func NewStore(dataDir string, id types.Hash) (*Store, error) {

	db, err := leveldb.OpenFile(dataDir, nil)

	if err != nil {
		return nil, err
	}

	store := &Store{
		memDb:                NewMemDB(),
		unconfirmedBatchMap:  make(map[types.Hash]*list.Element),
		unconfirmedBatchList: list.New(),

		snapshotMemDb: NewSnapshotMemDB(),

		flushingBatch: new(leveldb.Batch),
		dbDir:         dataDir,
		db:            db,
		id:            id,
	}

	return store, nil
}

func (store *Store) NewBatch() *leveldb.Batch {
	return new(leveldb.Batch)
}

func (store *Store) WriteDirectly(batch *leveldb.Batch) {
	store.mu.Lock()
	defer store.mu.Unlock()

	batch.Replay(store.memDb)
	batch.Replay(store.snapshotMemDb)
}

func (store *Store) WriteAccountBlock(batch *leveldb.Batch, block *ledger.AccountBlock) {
	store.WriteAccountBlockByHash(batch, block.Hash)
}

func (store *Store) WriteAccountBlockByHash(batch *leveldb.Batch, blockHash types.Hash) {
	//return store.cache.WriteAccountBlock(batch, nil)
	store.mu.Lock()
	defer store.mu.Unlock()

	// write store.unconfirmedBatch
	elem := store.unconfirmedBatchList.PushBack(batch)
	store.unconfirmedBatchMap[blockHash] = elem

	// write store.memDb
	batch.Replay(store.memDb)
}

// snapshot
func (store *Store) WriteSnapshot(snapshotBatch *leveldb.Batch, accountBlock []*ledger.AccountBlock) {
	store.mu.Lock()
	defer store.mu.Unlock()

	// write store.memDb
	if snapshotBatch != nil {
		snapshotBatch.Replay(store.memDb)
	}

	for _, block := range accountBlock {
		if elem, ok := store.unconfirmedBatchMap[block.Hash]; ok {
			// patch
			elem.Value.(*leveldb.Batch).Replay(store.snapshotMemDb)
			// remove
			delete(store.unconfirmedBatchMap, block.Hash)
			store.unconfirmedBatchList.Remove(elem)
		}
	}

	// write store snapshot memDb
	if snapshotBatch != nil {
		snapshotBatch.Replay(store.snapshotMemDb)
	}
}

// snapshot
func (store *Store) WriteSnapshotByHash(snapshotBatch *leveldb.Batch, blockHashList []types.Hash) {
	store.mu.Lock()
	defer store.mu.Unlock()

	// write store.memDb
	if snapshotBatch != nil {
		snapshotBatch.Replay(store.memDb)
	}

	for _, blockHash := range blockHashList {
		if elem, ok := store.unconfirmedBatchMap[blockHash]; ok {
			elem.Value.(*leveldb.Batch).Replay(store.snapshotMemDb)
			// delete
			delete(store.unconfirmedBatchMap, blockHash)
			store.unconfirmedBatchList.Remove(elem)
		}
	}

	// write store snapshot memDb
	if snapshotBatch != nil {
		snapshotBatch.Replay(store.snapshotMemDb)
	}
}

// rollback
func (store *Store) RollbackAccountBlocks(rollbackBatch *leveldb.Batch, accountBlocks []*ledger.AccountBlock) {
	store.mu.Lock()
	defer store.mu.Unlock()

	// delete store.unconfirmedBatchMap
	for _, block := range accountBlocks {
		if elem, ok := store.unconfirmedBatchMap[block.Hash]; ok {
			// delete
			delete(store.unconfirmedBatchMap, block.Hash)
			store.unconfirmedBatchList.Remove(elem)
		}

	}

	// write store.memDb
	rollbackBatch.Replay(store.memDb)
}

// rollback
func (store *Store) RollbackAccountBlockByHash(rollbackBatch *leveldb.Batch, blockHashList []types.Hash) {
	store.mu.Lock()
	defer store.mu.Unlock()

	// delete store.unconfirmedBatchMap
	for _, blockHash := range blockHashList {
		if elem, ok := store.unconfirmedBatchMap[blockHash]; ok {
			// delete
			delete(store.unconfirmedBatchMap, blockHash)
			store.unconfirmedBatchList.Remove(elem)
		}
	}

	// write store.memDb
	rollbackBatch.Replay(store.memDb)
}

func (store *Store) RollbackSnapshot(rollbackBatch *leveldb.Batch) {
	store.mu.Lock()
	defer store.mu.Unlock()

	// write store.memDb
	rollbackBatch.Replay(store.memDb)

	// copy
	copyBatch := new(leveldb.Batch)
	store.memDb.Flush(copyBatch)

	store.snapshotMemDb.Reset()
	copyBatch.Replay(store.snapshotMemDb)

	// reset
	store.unconfirmedBatchList = new(list.List)
	store.unconfirmedBatchMap = make(map[types.Hash]*list.Element)

}

func (store *Store) Get(key []byte) ([]byte, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	value, ok := store.memDb.Get(key)
	if !ok {
		var err error
		value, err = store.db.Get(key, nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
	}

	return value, nil
}

func (store *Store) Has(key []byte) (bool, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if ok, deleted := store.memDb.Has(key); ok {
		return ok, nil

	} else if deleted {
		return false, nil

	}

	return store.db.Has(key, nil)
}

func (store *Store) HasPrefix(prefix []byte) (bool, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	iter := store.newIterator(util.BytesPrefix(prefix))
	defer iter.Release()

	result := false
	for iter.Next() {
		result = true
		break
	}

	if err := iter.Error(); err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}
		return false, err
	}

	return result, nil
}

func (store *Store) NewIterator(slice *util.Range) interfaces.StorageIterator {
	store.mu.RLock()
	defer store.mu.RUnlock()

	return store.newIterator(slice)
}

func (store *Store) Close() error {
	store.mu.Lock()
	defer store.mu.Unlock()

	return store.close()
}

func (store *Store) Clean() error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if err := store.close(); err != nil {
		return err
	}

	if err := os.RemoveAll(store.dbDir); err != nil && err != os.ErrNotExist {
		return errors.New("Remove " + store.dbDir + " failed, error is " + err.Error())
	}

	store.db = nil

	return nil
}

func (store *Store) newIterator(slice *util.Range) interfaces.StorageIterator {
	return dbutils.NewMergedIterator([]interfaces.StorageIterator{
		store.memDb.NewIterator(slice),
		store.db.NewIterator(slice, nil),
	}, store.memDb.IsDelete)
}

func (store *Store) close() error {
	store.memDb = nil
	return store.db.Close()
}
