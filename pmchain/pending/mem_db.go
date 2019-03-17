package chain_pending

import (
	"github.com/vitelabs/go-vite/common/types"
	"sync"
)

type MemDB struct {
	mu          sync.RWMutex
	storage     map[string][]byte
	hashKeyList map[types.Hash][][]byte
}

func NewMemDB() *MemDB {
	return &MemDB{
		storage:     make(map[string][]byte),
		hashKeyList: make(map[types.Hash][][]byte),
	}
}

func (mDb *MemDB) Put(blockHash *types.Hash, key, value []byte) {
	mDb.mu.Lock()
	defer mDb.mu.Unlock()

	mDb.storage[string(key)] = value
	mDb.hashKeyList[*blockHash] = append(mDb.hashKeyList[*blockHash], key)
}

func (mDb *MemDB) Get(key []byte) ([]byte, bool) {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	result, ok := mDb.storage[string(key)]
	return result, ok
}

func (mDb *MemDB) Has(key []byte) bool {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	_, ok := mDb.storage[string(key)]
	return ok
}

func (mDb *MemDB) GetByBlockHash(blockHash *types.Hash) ([][]byte, [][]byte) {
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

func (mDb *MemDB) GetByBlockHashList(blockHashList []*types.Hash) ([][]byte, [][]byte) {
	mDb.mu.RLock()
	defer mDb.mu.RUnlock()

	size := 0
	for _, blockHash := range blockHashList {
		size += len(mDb.hashKeyList[*blockHash])
	}
	// less memory copy
	keyList := make([][]byte, 0, size)
	valueList := make([][]byte, 0, size)

	for _, blockHash := range blockHashList {
		keyList = append(keyList, mDb.hashKeyList[*blockHash]...)
		if len(keyList) <= 0 {
			return nil, nil
		}

		for key := range keyList {
			valueList = append(valueList, mDb.storage[string(key)])
		}

	}

	return keyList, valueList
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
