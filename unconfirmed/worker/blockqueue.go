package worker

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"sync"
)

var Vite_TokenId = "vite"

type BlockQueue struct {
	items       []*ledger.AccountBlock
	totalBalnce *big.Int
	fromName    *types.Address
	lock        sync.RWMutex
}

func (q *BlockQueue) Dequeue() *ledger.AccountBlock {
	q.lock.Lock()
	defer q.lock.Unlock()
	item := q.items[0]
	q.items = q.items[1:len(q.items)]

	if q.totalBalnce != nil {
		q.totalBalnce.Sub(q.totalBalnce, item.Balance[Vite_TokenId])
	}
	return item
}

func (q *BlockQueue) Enqueue(block *ledger.AccountBlock) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.items = append(q.items, block)
	if q.totalBalnce == nil {
		q.totalBalnce = &big.Int{}
	}
	q.totalBalnce.Add(q.totalBalnce, block.Balance[Vite_TokenId])
}

func (q *BlockQueue) Front() *ledger.AccountBlock {
	q.lock.Lock()
	defer q.lock.Unlock()
	item := q.items[0]
	return item
}

func (q *BlockQueue) Size() int {
	return len(q.items)
}

func (q *BlockQueue) IsEmpty() bool {
	return len(q.items) == 0
}

func (q *BlockQueue) Clear() {
	if cap(q.items) > 0 {
		q.items = q.items[:cap(q.items)]
	}
}

// InsertNew is sorted by block Height
func (q *BlockQueue) InsertNew(block *ledger.AccountBlock) {
	q.lock.Lock()
	defer q.lock.Unlock()
	for k, v := range q.items {
		if block.Height > v.Height {
			newSlice := q.items[0 : k-1]
			newSlice = append(newSlice, block)
		}
	}
	if q.totalBalnce == nil {
		q.totalBalnce = &big.Int{}
	}
	q.totalBalnce.Add(q.totalBalnce, block.Balance[Vite_TokenId])
}
