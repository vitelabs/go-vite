package queue

import (
	"sync"

	"github.com/vitelabs/go-vite/tools/list"
)

// Queue is the FIFO data structure
type Queue interface {
	// Add a element to the list tail, return false if list is full or closed
	Add(v interface{}) bool
	// Poll retrieve the element at list head
	Poll() interface{}
	// Size return the count of elements in queue
	Size() int
	// Close signal blockQueue list to avoid endless wait
	Close()
}

type blockQueue struct {
	mu     *sync.Mutex
	cond   *sync.Cond
	list   list.List
	closed bool
	max    int
}

// NewBlockQueue construct a block queue, max is the maximum elements can add into queue.
func NewBlockQueue(max int) Queue {
	mu := new(sync.Mutex)
	return &blockQueue{
		mu:   mu,
		cond: &sync.Cond{L: mu},
		list: list.New(),
		max:  max,
	}
}

func (q *blockQueue) Poll() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.list.Size() == 0 && !q.closed {
		q.cond.Wait()
	}

	return q.list.Shift()
}

// Add return true if add success, else return false.
func (q *blockQueue) Add(v interface{}) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed && q.list.Size() < q.max {
		q.list.Append(v)
		q.cond.Signal()

		return true
	}

	return false
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
		q.cond.Signal()
	}
}
