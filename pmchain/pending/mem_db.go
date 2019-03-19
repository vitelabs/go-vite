package chain_pending

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"sync"
)

const (
	valueMark   = byte(0)
	deletedMark = byte(1)
)

type MemDB struct {
	mu        sync.RWMutex
	dataStore interfaces.Store

	storage     map[string][]byte
	hashKeyList map[types.Hash][][]byte
}

func NewMemDB(dataStore interfaces.Store) *MemDB {
	return &MemDB{
		dataStore:   dataStore,
		storage:     make(map[string][]byte),
		hashKeyList: make(map[types.Hash][][]byte),
	}
}

func (mDb *MemDB) Append(blockHash *types.Hash, key, value []byte) error {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	innerValue, ok := mDb.storage[string(key)]
	if !ok {
		if mDb.dataStore != nil {
			if ok, err := mDb.dataStore.Has(key); err != nil {
				return err
			} else if ok {
				dsValue, err := mDb.dataStore.Get(key)
				if err != nil {
					return err
				}

				value = append(dsValue, value...)
			}
		}

		mDb.put(blockHash, key, value)
		return nil
	}
	innerValue = append(innerValue, value...)

	mDb.storage[string(key)] = innerValue
	return nil
}

func (mDb *MemDB) Put(blockHash *types.Hash, key, value []byte) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()
	mDb.put(blockHash, key, value)
}

func (mDb *MemDB) Get(key []byte) ([]byte, bool) {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	result, ok := mDb.storage[string(key)]
	if result[0] == deletedMark {
		return nil, true
	}
	return result[1:], ok
}

func (mDb *MemDB) Has(key []byte) bool {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	value, ok := mDb.storage[string(key)]
	if !ok {
		return false
	}
	if value[0] == deletedMark {
		return false
	}

	return true
}

func (mDb *MemDB) Flush(batch interfaces.Batch, blockHashList []*types.Hash) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	for _, blockHash := range blockHashList {
		keyList := mDb.hashKeyList[*blockHash]
		for _, key := range keyList {
			value := mDb.storage[string(key)]
			if value[0] == deletedMark {
				batch.Delete(key)
			} else {
				batch.Put(key, value[1:])
			}
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

	mDb.storage = make(map[string][]byte)
	mDb.hashKeyList = make(map[types.Hash][][]byte)
}

func (mDb *MemDB) put(blockHash *types.Hash, key, value []byte) {

	innerValue := make([]byte, len(value)+1)
	copy(innerValue[1:], value)

	mDb.storage[string(key)] = innerValue
	mDb.hashKeyList[*blockHash] = append(mDb.hashKeyList[*blockHash], key)
}

func (mDb *MemDB) deleteByBlockHash(blockHash *types.Hash) {
	keyList := mDb.hashKeyList[*blockHash]
	if len(keyList) <= 0 {
		return
	}

	for key := range keyList {
		delete(mDb.storage, string(key))
	}
	delete(mDb.hashKeyList, *blockHash)
}
