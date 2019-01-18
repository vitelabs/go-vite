package blockQueue

import (
	"sync"

	"github.com/vitelabs/go-vite/p2p/list"
)

type BlockQueue interface {
	Push(v interface{})
	Pop() interface{}
	Size() int
	Close()
}

type blockQueue struct {
	mu     *sync.Mutex
	cond   *sync.Cond
	list   list.List
	closed bool
}

func New() BlockQueue {
	mu := new(sync.Mutex)
	return &blockQueue{
		mu:   mu,
		cond: &sync.Cond{L: mu},
		list: list.New(),
	}
}

func (q *blockQueue) Pop() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.list.Size() == 0 && !q.closed {
		q.cond.Wait()
	}

	return q.list.Shift()
}

func (q *blockQueue) Push(v interface{}) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.list.Append(v)
		q.cond.Broadcast()
	}
}

func (q *blockQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.list.Size()
}

func (q *blockQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		q.cond.Broadcast()

	}
}
