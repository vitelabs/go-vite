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

func NewUnconfirmedBatchs() *UnconfirmedBatchs {
	return &UnconfirmedBatchs{
		batchMap:  make(map[types.Hash]*list.Element),
		batchList: list.New(),
	}
}

func (ub *UnconfirmedBatchs) Size() int {
	ub.mu.RLock()
	defer ub.mu.RUnlock()

	return len(ub.batchMap)
}

func (ub *UnconfirmedBatchs) Get(blockHash types.Hash) (*leveldb.Batch, bool) {
	ub.mu.RLock()
	defer ub.mu.RUnlock()

	if elem, ok := ub.batchMap[blockHash]; ok {
		return elem.Value.(*leveldb.Batch), ok
	}
	return nil, false
}

func (ub *UnconfirmedBatchs) Put(blockHash types.Hash, batch *leveldb.Batch) {
	ub.mu.Lock()
	defer ub.mu.Unlock()

	elem := ub.batchList.PushBack(batch)
	ub.batchMap[blockHash] = elem
}

func (ub *UnconfirmedBatchs) Remove(blockHash types.Hash) {
	ub.mu.Lock()
	defer ub.mu.Unlock()

	if elem, ok := ub.batchMap[blockHash]; ok {
		ub.batchList.Remove(elem)
		delete(ub.batchMap, blockHash)
	}
}

func (ub *UnconfirmedBatchs) Clear() {
	ub.mu.Lock()
	defer ub.mu.Unlock()

	ub.batchMap = make(map[types.Hash]*list.Element)
	ub.batchList = list.New()
}

func (ub *UnconfirmedBatchs) All(f func(batch *leveldb.Batch)) {
	ub.mu.RLock()
	defer ub.mu.RUnlock()
	elem := ub.batchList.Front()
	for elem != nil {
		f(elem.Value.(*leveldb.Batch))
		elem = elem.Next()
	}

}
