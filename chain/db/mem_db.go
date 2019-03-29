package chain_db

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"sync"
)

type MemDB struct {
	mu sync.RWMutex

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

func (mDb *MemDB) Lock() {
	mDb.mu.Lock()
}

func (mDb *MemDB) Unlock() {
	mDb.mu.Unlock()
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

	delete(mDb.deletedKey, string(key))
}

func (mDb *MemDB) Has(key []byte) (bool, deleted bool) {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()
	if _, ok := mDb.deletedKey[string(key)]; ok {
		return false, true
	}

	return mDb.storage.Contains(key), false
}

func (mDb *MemDB) HasByPrefix(prefixKey []byte) bool {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

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

//func (mDb *MemDB) deleteByBlockHash(blockHash *types.Hash) {
//	keyList, ok := mDb.hashKeyMap[*blockHash]
//
//	if !ok {
//		return
//	}
//
//	for keyStr := range keyList {
//		delete(mDb.deletedKey, keyStr)
//		mDb.storage.Delete([]byte(keyStr))
//	}
//	delete(mDb.hashKeyMap, *blockHash)
//}

//package chain_db
//
//import (
//	"bytes"
//	"github.com/syndtr/goleveldb/leveldb/comparer"
//	"github.com/syndtr/goleveldb/leveldb/iterator"
//	"github.com/syndtr/goleveldb/leveldb/memdb"
//	"github.com/syndtr/goleveldb/leveldb/util"
//	"github.com/vitelabs/go-vite/common/types"
//	"github.com/vitelabs/go-vite/interfaces"
//	"sync"
//)
//
//type MemDB struct {
//	mu sync.RWMutex
//
//	storage *memdb.DB
//
//	deletedKey map[string]struct{}
//	hashKeyMap map[types.Hash]map[string]struct{}
//}
//
//func NewMemDB() *MemDB {
//	return &MemDB{
//		storage:    newStorage(),
//		deletedKey: make(map[string]struct{}),
//		hashKeyMap: make(map[types.Hash]map[string]struct{}),
//	}
//}
//
//func newStorage() *memdb.DB {
//	return memdb.New(comparer.DefaultComparer, 10*1024*1024)
//}
//
//func (mDb *MemDB) IsDelete(key []byte) bool {
//	mDb.mu.RLock()
//	defer mDb.mu.RUnlock()
//
//	_, ok := mDb.deletedKey[string(key)]
//	return ok
//}
//
//func (mDb *MemDB) NewIterator(slice *util.Range) iterator.Iterator {
//	return mDb.storage.NewIterator(slice)
//}
//
//func (mDb *MemDB) Put(blockHash *types.Hash, key, value []byte) {
//	mDb.mu.Lock()
//	defer mDb.mu.Unlock()
//	mDb.put(blockHash, key, value)
//}
//
//func (mDb *MemDB) Get(key []byte) ([]byte, bool) {
//	mDb.mu.RLock()
//	defer mDb.mu.RUnlock()
//	return mDb.get(key)
//}
//
//func (mDb *MemDB) Delete(blockHash *types.Hash, key []byte) {
//	mDb.mu.Lock()
//	defer mDb.mu.Unlock()
//
//	mDb.storage.Delete(key)
//
//	strKey := string(key)
//
//	mDb.deletedKey[string(key)] = struct{}{}
//	delete(mDb.hashKeyMap[*blockHash], strKey)
//}
//
//func (mDb *MemDB) Has(key []byte) (bool, deleted bool) {
//	mDb.mu.RLock()
//	defer mDb.mu.RUnlock()
//	if _, ok := mDb.deletedKey[string(key)]; ok {
//		return false, true
//	}
//
//	return mDb.storage.Contains(key), false
//}
//
//func (mDb *MemDB) HasByPrefix(prefixKey []byte) bool {
//	mDb.mu.RLock()
//	defer mDb.mu.RUnlock()
//
//	key, _, errNotFound := mDb.storage.Find(prefixKey)
//	if errNotFound != nil {
//		return false
//	}
//
//	return bytes.HasPrefix(key, prefixKey)
//}
//
//func (mDb *MemDB) FlushList(batch interfaces.Batch, blockHashList []*types.Hash) {
//	mDb.mu.Lock()
//	defer mDb.mu.Unlock()
//
//	for _, blockHash := range blockHashList {
//		keyMap := mDb.hashKeyMap[*blockHash]
//		for keyStr := range keyMap {
//			key := []byte(keyStr)
//			if _, ok := mDb.deletedKey[keyStr]; ok {
//				delete(mDb.deletedKey, keyStr)
//				batch.Delete(key)
//			} else {
//				value, ok := mDb.get(key)
//				if !ok {
//					continue
//				}
//
//				batch.Put(key, value)
//			}
//
//		}
//	}
//}
//
//func (mDb *MemDB) Flush(batch interfaces.Batch, blockHash *types.Hash) {
//	mDb.mu.Lock()
//	defer mDb.mu.Unlock()
//
//	keyMap := mDb.hashKeyMap[*blockHash]
//	for keyStr := range keyMap {
//		key := []byte(keyStr)
//		if _, ok := mDb.deletedKey[keyStr]; ok {
//			delete(mDb.deletedKey, keyStr)
//			batch.Delete(key)
//		} else {
//			value, ok := mDb.get(key)
//			if !ok {
//				continue
//			}
//
//			batch.Put(key, value)
//		}
//
//	}
//}
//
//func (mDb *MemDB) DeleteByBlockHash(blockHash *types.Hash) {
//	mDb.mu.Lock()
//	defer mDb.mu.Unlock()
//
//	mDb.deleteByBlockHash(blockHash)
//}
//
//func (mDb *MemDB) DeleteByBlockHashList(blockHashList []*types.Hash) {
//	mDb.mu.Lock()
//	defer mDb.mu.Unlock()
//
//	for _, blockHsh := range blockHashList {
//		mDb.deleteByBlockHash(blockHsh)
//	}
//}
//
//func (mDb *MemDB) Clean() {
//	mDb.mu.Lock()
//	defer mDb.mu.Unlock()
//
//	mDb.storage.Reset()
//
//	mDb.deletedKey = make(map[string]struct{})
//
//	mDb.hashKeyMap = make(map[types.Hash]map[string]struct{})
//}
//
//func (mDb *MemDB) get(key []byte) ([]byte, bool) {
//	if _, ok := mDb.deletedKey[string(key)]; ok {
//		return nil, true
//	}
//
//	result, errNotFound := mDb.storage.Get(key)
//	if errNotFound != nil {
//		return nil, false
//	}
//	return result, true
//}
//
//func (mDb *MemDB) put(blockHash *types.Hash, key, value []byte) {
//	keyStr := string(key)
//	if _, ok := mDb.deletedKey[keyStr]; ok {
//		delete(mDb.deletedKey, keyStr)
//	}
//	if _, ok := mDb.hashKeyMap[*blockHash]; !ok {
//		mDb.hashKeyMap[*blockHash] = make(map[string]struct{})
//	}
//	mDb.storage.Put(key, value)
//
//	mDb.hashKeyMap[*blockHash][keyStr] = struct{}{}
//}
//
//func (mDb *MemDB) deleteByBlockHash(blockHash *types.Hash) {
//	keyList, ok := mDb.hashKeyMap[*blockHash]
//
//	if !ok {
//		return
//	}
//
//	for keyStr := range keyList {
//		delete(mDb.deletedKey, keyStr)
//		mDb.storage.Delete([]byte(keyStr))
//	}
//	delete(mDb.hashKeyMap, *blockHash)
//}
