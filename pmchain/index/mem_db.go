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

func (mDb *memDb) GetAndDelete(blockHash *types.Hash) ([][]byte, [][]byte) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	keyList := mDb.hashKeyList[*blockHash]
	if len(keyList) <= 0 {
		return nil, nil
	}

	valueList := make([][]byte, 0, len(keyList))
	for key := range keyList {
		keyString := string(key)
		valueList = append(valueList, mDb.storage[keyString])

		delete(mDb.storage, keyString)
	}
	delete(mDb.hashKeyList, *blockHash)

	return keyList, valueList
}
