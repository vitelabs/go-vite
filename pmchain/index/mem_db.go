package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"sync"
)

type memDb struct {
	mu          sync.RWMutex
	storage     map[string][]byte
	hashKeyList map[types.Hash][][]byte
}

func newMemDb() MemDB {
	return &memDb{
		storage:     make(map[string][]byte),
		hashKeyList: make(map[types.Hash][][]byte),
	}
}

func (mDb *memDb) Put(blockHash *types.Hash, key, value []byte) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	mDb.storage[string(key)] = value
	mDb.hashKeyList[*blockHash] = append(mDb.hashKeyList[*blockHash], key)
}

func (mDb *memDb) Get(key []byte) ([]byte, bool) {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	result, ok := mDb.storage[string(key)]
	return result, ok
}

func (mDb *memDb) GetByBlockHash(blockHash *types.Hash) ([][]byte, [][]byte) {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	keyList := mDb.hashKeyList[*blockHash]
	if len(keyList) <= 0 {
		return nil, nil
	}

	valueList := make([][]byte, 0, len(keyList))
	for key := range keyList {
		valueList = append(valueList, mDb.storage[string(key)])
	}

	return keyList, valueList
}

func (mDb *memDb) DeleteByBlockHash(blockHash *types.Hash) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	keyList := mDb.hashKeyList[*blockHash]
	if len(keyList) <= 0 {
		return
	}

	for key := range keyList {
		delete(mDb.storage, string(key))
	}
	delete(mDb.hashKeyList, *blockHash)
}

func (mDb *memDb) Clean() {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	mDb.storage = make(map[string][]byte)
	mDb.hashKeyList = make(map[types.Hash][][]byte)
}
