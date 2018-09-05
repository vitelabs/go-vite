package worker

import (
	"github.com/vitelabs/go-vite/unconfirmed"
	"sync"
)

type BlockQueue struct {
	items []*unconfirmed.AccountBlock
	lock  sync.RWMutex
	//totalBalnce *big.Int
}

// Enqueue is sorted by block Height
func (q *BlockQueue) Enqueue(block *unconfirmed.AccountBlock) {
	q.lock.Lock()
	defer q.lock.Unlock()
	for k, v := range q.items {
		if block.Height.Cmp(v.Height) == -1 {
			newSlice := q.items[0 : k-1]
			newSlice = append(newSlice, block)
		}
	}
	//if q.totalBalnce == nil {
	//	q.totalBalnce = &big.Int{}
	//}
	//q.totalBalnce.Add(q.totalBalnce, block.Balance[Vite_TokenId])
}

func (q *BlockQueue) Dequeue() *unconfirmed.AccountBlock {
	q.lock.Lock()
	defer q.lock.Unlock()
	item := q.items[0]
	q.items = q.items[1:len(q.items)]

	//if q.totalBalnce != nil {
	//	q.totalBalnce.Sub(q.totalBalnce, item.Balance[Vite_TokenId])
	//}
	return item
}

func (q *BlockQueue) Front() *unconfirmed.AccountBlock {
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
