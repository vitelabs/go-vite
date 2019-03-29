package chain_db

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type MemDB struct {
	storage *memdb.DB

	deletedKey map[string]struct{}
}

func NewMemDB() *MemDB {
	return &MemDB{
		storage:    newStorage(),
		deletedKey: make(map[string]struct{}),
	}
}

func newStorage() *memdb.DB {
	return memdb.New(comparer.DefaultComparer, 10*1024*1024)
}

func (mDb *MemDB) IsDelete(key []byte) bool {
	_, ok := mDb.deletedKey[string(key)]
	return ok
}

func (mDb *MemDB) NewIterator(slice *util.Range) iterator.Iterator {
	return mDb.storage.NewIterator(slice)
}

func (mDb *MemDB) Put(key, value []byte) {
	mDb.put(key, value)
}

func (mDb *MemDB) Get(key []byte) ([]byte, bool) {
	return mDb.get(key)
}

func (mDb *MemDB) Delete(key []byte) {
	mDb.storage.Delete(key)

	delete(mDb.deletedKey, string(key))
}

func (mDb *MemDB) Has(key []byte) (bool, deleted bool) {
	if _, ok := mDb.deletedKey[string(key)]; ok {
		return false, true
	}

	return mDb.storage.Contains(key), false
}

func (mDb *MemDB) HasByPrefix(prefixKey []byte) bool {

	key, _, errNotFound := mDb.storage.Find(prefixKey)
	if errNotFound != nil {
		return false
	}

	return bytes.HasPrefix(key, prefixKey)
}

func (mDb *MemDB) Flush(batch *leveldb.Batch) error {
	iter := mDb.storage.NewIterator(nil)
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

func (mDb *MemDB) Clean() {
	mDb.deletedKey = make(map[string]struct{})
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
