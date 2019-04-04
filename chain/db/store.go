package chain_db

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"os"
	"sync"
	"time"
)

type Store struct {
	id    types.Hash
	mu    sync.RWMutex
	memDb *MemDB

	dbDir string
	db    *leveldb.DB

	flushingBatch *leveldb.Batch
}

func NewStore(dataDir string, flushInterval time.Duration, id types.Hash) (*Store, error) {
	if flushInterval <= 0 {
		flushInterval = time.Second
	}

	db, err := leveldb.OpenFile(dataDir, nil)

	if err != nil {
		return nil, err
	}

	store := &Store{
		memDb:         NewMemDB(),
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

func (store *Store) Write(batch *leveldb.Batch) {
	//return store.db.Write(batch, nil)
	store.mu.Lock()
	defer store.mu.Unlock()

	batch.Replay(store.memDb)
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

	iter := store.NewIterator(util.BytesPrefix(prefix))
	defer iter.Release()

	result := false
	for iter.Next() {
		result = true
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
	return dbutils.NewMergedIterator([]interfaces.StorageIterator{
		store.memDb.NewIterator(slice),
		store.db.NewIterator(slice, nil),
	}, store.memDb.IsDelete)
}

func (store *Store) Close() error {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.memDb = nil
	return store.db.Close()
}

func (store *Store) Clean() error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if err := store.Close(); err != nil {
		return err
	}

	if err := os.RemoveAll(store.dbDir); err != nil && err != os.ErrNotExist {
		return errors.New("Remove " + store.dbDir + " failed, error is " + err.Error())
	}

	store.db = nil

	return nil
}

func (store *Store) Id() types.Hash {
	return store.id
}

func (store *Store) Prepare() {
	store.mu.RLock()
	defer store.mu.RUnlock()

	store.flushingBatch.Reset()
	store.memDb.Flush(store.flushingBatch)
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
	batch.Load(redoLog)

	if err := store.db.Write(batch, nil); err != nil {
		return err
	}

	store.resetMemDB()
	return nil
}

func (store *Store) resetMemDB() {
	store.mu.Lock()
	store.memDb = NewMemDB()
	store.mu.Unlock()
}
