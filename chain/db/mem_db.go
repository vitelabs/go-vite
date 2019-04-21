package chain_db

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"sync"
)

type MemDB struct {
	storage *memdb.DB

	deletedKey map[string]struct{} // for deleted key
	mu         sync.RWMutex
}

func NewMemDB() *MemDB {
	return &MemDB{
		storage:    newStorage(),
		deletedKey: make(map[string]struct{}),
	}
}

func newStorage() *memdb.DB {
	return memdb.New(comparer.DefaultComparer, 0)
}

func (mDb *MemDB) IsDelete(key []byte) bool {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	_, ok := mDb.deletedKey[string(key)]
	return ok
}

func (mDb *MemDB) NewIterator(slice *util.Range) iterator.Iterator {
	return mDb.storage.NewIterator(slice)
}

func (mDb *MemDB) Put(key, value []byte) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	mDb.put(key, value)
}

func (mDb *MemDB) Get(key []byte) ([]byte, bool) {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	return mDb.get(key)
}

func (mDb *MemDB) Delete(key []byte) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	mDb.storage.Delete(key)

	mDb.deletedKey[string(key)] = struct{}{}

}

func (mDb *MemDB) Has(key []byte) (bool, deleted bool) {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	if _, ok := mDb.deletedKey[string(key)]; ok {
		return false, true
	}

	return mDb.storage.Contains(key), false
}

func (mDb *MemDB) Flush(batch *leveldb.Batch) error {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	iter := mDb.storage.NewIterator(nil)
	defer iter.Release()
	for iter.Next() {
		batch.Put(iter.Key(), iter.Value())
	}

	if err := iter.Error(); err != nil {
		return err
	}
	for key := range mDb.deletedKey {
		batch.Delete([]byte(key))
	}

	return nil

}

func (mDb *MemDB) get(key []byte) ([]byte, bool) {

	if _, ok := mDb.deletedKey[string(key)]; ok {
		return nil, true
	}

	result, errNotFound := mDb.storage.Get(key)
	if errNotFound != nil {
		return nil, false
	}
	return result, true
}

func (mDb *MemDB) put(key, value []byte) {
	keyStr := string(key)
	if _, ok := mDb.deletedKey[keyStr]; ok {
		delete(mDb.deletedKey, keyStr)
	}

	mDb.storage.Put(key, value)
}
