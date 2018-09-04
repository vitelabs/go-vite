package worker

import (
	"container/heap"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"github.com/vitelabs/go-vite/unconfirmed"
)

type fromItem struct {
	key      *types.Address // The key of the fromItem, sendBlock fromAddress
	value    *BlockQueue    // The value of the fromItem; arbitrary.
	priority *big.Int       // The priority of the fromItem in the queue.
	index    int            // The index of the fromItem in the heap.
}

type PriorityFromQueue []*fromItem

func (pfq *PriorityFromQueue) Push(x interface{}) {
	item := x.(*fromItem)
	item.index = len(*pfq)
	*pfq = append(*pfq, item)
}

func (pfq *PriorityFromQueue) Pop() interface{} {
	old := *pfq
	n := len(old)
	fItem := old[n-1]
	fItem.index = -1 // for safety
	*pfq = old[0 : n-1]
	return fItem
}

func (pfq *PriorityFromQueue) Len() int { return len(*pfq) }

func (pfq *PriorityFromQueue) Less(i, j int) bool {
	if (*pfq)[i].priority.Cmp((*pfq)[j].priority) == -1 {
		return true
	} else {
		return false
	}
}

func (pfq *PriorityFromQueue) Swap(i, j int) {
	(*pfq)[i], (*pfq)[j] = (*pfq)[j], (*pfq)[i]
	(*pfq)[i].index = i
	(*pfq)[j].index = j
}

func (pfq *PriorityFromQueue) InsertNew(block *unconfirmed.AccountBlock) {
	var find = false
	for _, v := range *pfq {
		if v.key == block.From {
			v.value.InsertNew(block)
			if v.priority == nil {
				v.priority = big.NewInt(0)
			}
			v.priority.Add(v.priority, block.Balance[Vite_TokenId])
			find = true
			break
		}
	}
	if !find {
		var bq BlockQueue
		bq.InsertNew(block)
		fItem := &fromItem{
			key:      block.From,
			value:    &bq,
			priority: block.Balance[Vite_TokenId],
		}
		pfq.Push(fItem)
	}
}

func (pfq *PriorityFromQueue) update(item *fromItem, key *types.Address, value *BlockQueue, priority *big.Int) {
	item.value = value
	item.priority = priority
	item.key = key
	heap.Fix(pfq, item.index)
}

type PriorityToQueue []*toItem

type toItem struct {
	key      *types.Address     // the key of the toItem, sendBlock toAddress
	value    *PriorityFromQueue // the value of the toItem; arbitrary.
	priority *big.Int           // the priority of the toItem in the queue.
	index    int                // the index of the toItem in the heap.
}

func (ptq *PriorityToQueue) Push(x interface{}) {
	item := x.(*toItem)
	item.index = len(*ptq)
	*ptq = append(*ptq, item)
}

func (ptq *PriorityToQueue) Pop() interface{} {
	old := *ptq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*ptq = old[0 : n-1]
	return item
}

func (ptq *PriorityToQueue) Len() int { return len(*ptq) }

func (ptq *PriorityToQueue) Less(i, j int) bool {
	if (*ptq)[i].priority.Cmp((*ptq)[j].priority) == -1 {
		return true
	} else {
		return false
	}
}

func (ptq *PriorityToQueue) Swap(i, j int) {
	(*ptq)[i], (*ptq)[j] = (*ptq)[j], (*ptq)[i]
	(*ptq)[i].index = i
	(*ptq)[j].index = j
}

func (ptq *PriorityToQueue) InsertNew(block *unconfirmed.AccountBlock) {
	var find = false
	for _, v := range *ptq {
		if v.key == block.To {
			v.value.InsertNew(block)
			if v.priority == nil {
				v.priority = big.NewInt(0)
			}
			v.priority.Add(v.priority, block.Balance[Vite_TokenId])
			find = true
			break
		}
	}
	if !find {
		var fQueue PriorityFromQueue
		fQueue.InsertNew(block)
		tItem := &toItem{
			key:   block.To,
			value: &fQueue,
			priority: block.Balance[Vite_TokenId],
		}
		ptq.Push(tItem)
	}
}

func (ptq *PriorityToQueue) update(item *toItem, key *types.Address, value *PriorityFromQueue, priority *big.Int) {
	item.value = value
	item.priority = priority
	item.key = key
	heap.Fix(ptq, item.index)
}
