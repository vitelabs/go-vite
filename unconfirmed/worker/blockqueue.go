package worker

import (
	"github.com/vitelabs/go-vite/ledger"
	"sync"
)
// thread safe
type BlockQueue struct {
	items        []*ledger.AccountBlock
	lock         sync.RWMutex
	pullListener chan struct{}
}

func NewBlockQueue() *BlockQueue {
	return nil
}

func (q *BlockQueue) PullFromMem() error {
	q.lock.Lock()
	// todo:  need to add rotation condition
	q.pullListener <- struct{}{}
	q.lock.Unlock()
	return nil
}

func (q *BlockQueue) Dequeue() *ledger.AccountBlock {
	q.lock.Lock()
	item := q.items[0]
	q.items = q.items[1:len(q.items)]
	q.lock.Unlock()
	return item
}

func (q *BlockQueue) Enqueue(block *ledger.AccountBlock) {
	q.lock.Lock()
	q.items = append(q.items, block)
	q.lock.Unlock()
}

func (q *BlockQueue) Front() *ledger.AccountBlock {
	q.lock.Lock()
	item := q.items[0]
	q.lock.Unlock()
	return item
}

func (q *BlockQueue) Size() int {
	return len(q.items)
}

func (q *BlockQueue) IsEmpty() bool {
	return len(q.items) == 0
}

func (q *BlockQueue) Clear() {

}