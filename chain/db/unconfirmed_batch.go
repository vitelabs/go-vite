package chain_db

import (
	"container/list"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/types"
	"sync"
)

type UnconfirmedBatchs struct {
	batchMap  map[types.Hash]*list.Element
	batchList *list.List
	mu        sync.RWMutex
}

func (ub *UnconfirmedBatchs) Get(blockHash types.Hash) *leveldb.Batch {
	ub.mu.RLock()
	defer ub.mu.RUnlock()

	if elem, ok := ub.batchMap[blockHash]; ok {
		return elem.(*leveldb.Batch)
	}
	return nil
}

func (ub *UnconfirmedBatchs) Remove(blockHash types.Hash) {
	ub.mu.Lock()
	defer ub.mu.Unlock()

	if batch, ok := ub.batchMap[blockHash]; ok {

	}
	return nil
}
