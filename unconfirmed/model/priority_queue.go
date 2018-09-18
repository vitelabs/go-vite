package model

import (
	"container/heap"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type FromItem struct {
	Key   types.Address // The key of the FromItem, sendBlock fromAddress
	Value *BlockQueue   // The value of the FromItem; arbitrary.

	Index    int    // The index of the FromItem in the heap.
	Priority uint64 // The priority of the FromItem in the queue.
}

type ToItem struct {
	Key   types.Address      // the key of the ToItem, sendBlock toAddress
	Value *PriorityFromQueue // the value of the ToItem; arbitrary.

	Index    int    // the index of the ToItem in the heap.
	Priority uint64 // the priority of the ToItem in the queue.
}

type PriorityFromQueue []*FromItem

func (pfq *PriorityFromQueue) Push(x interface{}) {
	item := x.(*FromItem)
	item.Index = len(*pfq)
	*pfq = append(*pfq, item)
}

func (pfq *PriorityFromQueue) Pop() interface{} {
	old := *pfq
	n := len(old)
	fItem := old[n-1]
	fItem.Index = -1 // for safety
	*pfq = old[0 : n-1]
	return fItem
}

func (pfq *PriorityFromQueue) Len() int { return len(*pfq) }

func (pfq *PriorityFromQueue) Less(i, j int) bool {
	if (*pfq)[i].Priority < (*pfq)[j].Priority {
		return true
	} else {
		return false
	}
}

func (pfq *PriorityFromQueue) Swap(i, j int) {
	(*pfq)[i], (*pfq)[j] = (*pfq)[j], (*pfq)[i]
	(*pfq)[i].Index = i
	(*pfq)[j].Index = j
}

func (pfq *PriorityFromQueue) InsertNew(block *ledger.AccountBlock, fromQuota uint64) {
	var find = false
	for _, v := range *pfq {
		if v.Key == block.AccountAddress {
			v.Value.Enqueue(block)
			v.Priority = fromQuota
			find = true
			break
		}
	}
	if !find {
		var bq BlockQueue
		bq.Enqueue(block)
		fItem := &FromItem{
			Key:   block.AccountAddress,
			Value: &bq,
		}
		fItem.Priority = fromQuota
		pfq.Push(fItem)
	}
}

func (pfq *PriorityFromQueue) update(item *FromItem, key *types.Address, value *BlockQueue, priority uint64) {
	item.Value = value
	item.Priority = priority
	item.Key = *key
	heap.Fix(pfq, item.Index)
}

func (pfq *PriorityFromQueue) Remove(key *types.Address) {
	for i := 0; i < pfq.Len(); i++ {
		if (*pfq)[i].Key == *key {
			heap.Remove(pfq, (*pfq)[i].Index)
		}
	}
}

type PriorityToQueue []*ToItem

func (ptq *PriorityToQueue) Push(x interface{}) {
	item := x.(*ToItem)
	item.Index = len(*ptq)
	*ptq = append(*ptq, item)
}

func (ptq *PriorityToQueue) Pop() interface{} {
	old := *ptq
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*ptq = old[0 : n-1]
	return item
}

func (ptq *PriorityToQueue) Len() int { return len(*ptq) }

func (ptq *PriorityToQueue) Less(i, j int) bool {
	if (*ptq)[i].Priority < (*ptq)[j].Priority {
		return true
	} else {
		return false
	}
}

func (ptq *PriorityToQueue) Swap(i, j int) {
	(*ptq)[i], (*ptq)[j] = (*ptq)[j], (*ptq)[i]
	(*ptq)[i].Index = i
	(*ptq)[j].Index = j
}

func (ptq *PriorityToQueue) InsertNew(block *ledger.AccountBlock, toQuota uint64, fromQuota uint64) {
	var find = false
	for _, v := range *ptq {
		if v.Key == block.ToAddress {
			v.Value.InsertNew(block, fromQuota)
			v.Priority = toQuota
			find = true
			break
		}
	}
	if !find {
		var fQueue PriorityFromQueue
		fQueue.InsertNew(block, fromQuota)
		tItem := &ToItem{
			Key:   block.ToAddress,
			Value: &fQueue,
		}
		tItem.Priority = toQuota
		ptq.Push(tItem)
	}
}

func (ptq *PriorityToQueue) update(item *ToItem, key *types.Address, value *PriorityFromQueue, priority uint64) {
	item.Value = value
	item.Priority = priority
	item.Key = *key
	heap.Fix(ptq, item.Index)
}
