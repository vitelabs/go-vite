package blockQueue

import (
	"github.com/vitelabs/go-vite/p2p/list"
	"sync"
)

type BlockQueue struct {
	mu     *sync.Mutex
	cond   *sync.Cond
	list   *list.List
	closed bool
}

func New() *BlockQueue {
	mu := new(sync.Mutex)
	return &BlockQueue{
		mu:   mu,
		cond: &sync.Cond{L: mu},
		list: list.New(),
	}
}

func (q *BlockQueue) Pop() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.list.Size() == 0 && !q.closed {
		q.cond.Wait()
	}

	return q.list.Shift()
}

func (q *BlockQueue) Push(v interface{}) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.list.Append(v)
		q.cond.Broadcast()
	}
}

func (q *BlockQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.list.Size()
}

func (q *BlockQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		q.cond.Broadcast()

	}
}
