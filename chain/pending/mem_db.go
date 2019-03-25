package chain_pending

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"sync"
)

type MemDB struct {
	mu sync.RWMutex

	storage *memdb.DB

	deletedKey  map[string]struct{}
	hashKeyList map[types.Hash][][]byte
}

func NewMemDB() *MemDB {
	return &MemDB{
		storage:     newStorage(),
		deletedKey:  make(map[string]struct{}),
		hashKeyList: make(map[types.Hash][][]byte),
	}
}

func newStorage() *memdb.DB {
	return memdb.New(comparer.DefaultComparer, 10*1024*1024)
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

func (mDb *MemDB) Put(blockHash *types.Hash, key, value []byte) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()
	mDb.put(blockHash, key, value)
}

func (mDb *MemDB) Get(key []byte) ([]byte, bool) {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()
	return mDb.get(key)
}

func (mDb *MemDB) Delete(blockHash *types.Hash, key []byte) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	mDb.storage.Delete(key)
	mDb.deletedKey[string(key)] = struct{}{}
	mDb.hashKeyList[*blockHash] = append(mDb.hashKeyList[*blockHash], key)
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

func (mDb *MemDB) FlushList(batch interfaces.Batch, blockHashList []*types.Hash) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	for _, blockHash := range blockHashList {
		keyList := mDb.hashKeyList[*blockHash]
		for _, key := range keyList {
			if _, ok := mDb.deletedKey[string(key)]; ok {
				delete(mDb.deletedKey, string(key))
				batch.Delete(key)
			} else {
				value, ok := mDb.get(key)
				if !ok {
					continue
				}

				batch.Put(key, value)
			}

		}
	}
}

func (mDb *MemDB) Flush(batch interfaces.Batch, blockHash *types.Hash) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	keyList := mDb.hashKeyList[*blockHash]
	for _, key := range keyList {
		if _, ok := mDb.deletedKey[string(key)]; ok {
			delete(mDb.deletedKey, string(key))
			batch.Delete(key)
		} else {
			value, ok := mDb.get(key)
			if !ok {
				continue
			}

			batch.Put(key, value)
		}

	}
}

func (mDb *MemDB) DeleteByBlockHash(blockHash *types.Hash) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	mDb.deleteByBlockHash(blockHash)
}

func (mDb *MemDB) DeleteByBlockHashList(blockHashList []*types.Hash) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	for _, blockHsh := range blockHashList {
		mDb.deleteByBlockHash(blockHsh)
	}
}

func (mDb *MemDB) Clean() {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	mDb.storage.Reset()

	mDb.deletedKey = make(map[string]struct{})

	mDb.hashKeyList = make(map[types.Hash][][]byte)
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

func (mDb *MemDB) put(blockHash *types.Hash, key, value []byte) {
	if _, ok := mDb.deletedKey[string(key)]; ok {
		delete(mDb.deletedKey, string(key))
	}
	mDb.storage.Put(key, value)
	mDb.hashKeyList[*blockHash] = append(mDb.hashKeyList[*blockHash], key)
}

func (mDb *MemDB) deleteByBlockHash(blockHash *types.Hash) {
	keyList := mDb.hashKeyList[*blockHash]
	if len(keyList) <= 0 {
		return
	}

	for _, key := range keyList {
		delete(mDb.deletedKey, string(key))
		mDb.storage.Delete(key)
	}
	delete(mDb.hashKeyList, *blockHash)
}
