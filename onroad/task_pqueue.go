package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
)

type contractTask struct {
	Addr  types.Address
	Index int
	Quota uint64
}

type contractTaskPQueue []*contractTask

func (q *contractTaskPQueue) Push(x interface{}) {
	item := x.(*contractTask)
	item.Index = len(*q)
	*q = append(*q, item)
}

func (q *contractTaskPQueue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*q = old[0 : n-1]
	return item
}

func (q *contractTaskPQueue) Len() int { return len(*q) }

func (q *contractTaskPQueue) Less(i, j int) bool {
	if (*q)[i].Quota < (*q)[j].Quota {
		return true
	} else {
		return false
	}
}

func (q *contractTaskPQueue) Swap(i, j int) {
	(*q)[i], (*q)[j] = (*q)[j], (*q)[i]
	(*q)[i].Index = i
	(*q)[j].Index = j
}
