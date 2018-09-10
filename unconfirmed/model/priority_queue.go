package model

import (
	"container/heap"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"github.com/vitelabs/go-vite/ledger"
)

var Vite_TokenId, _ = types.BytesToTokenTypeId([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

type FromItem struct {
	Key   *types.Address // The key of the FromItem, sendBlock fromAddress
	Value *BlockQueue    // The value of the FromItem; arbitrary.

	Index    int      // The index of the FromItem in the heap.
	Priority *big.Int // The priority of the FromItem in the queue.
}

type ToItem struct {
	Key   *types.Address     // the key of the ToItem, sendBlock toAddress
	Value *PriorityFromQueue // the value of the ToItem; arbitrary.

	Index    int      // the index of the ToItem in the heap.
	Priority *big.Int // the priority of the ToItem in the queue.
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
	if (*pfq)[i].Priority.Cmp((*pfq)[j].Priority) == -1 {
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

func (pfq *PriorityFromQueue) InsertNew(block *ledger.AccountBlock) {
	var find = false
	for _, v := range *pfq {
		if v.Key == block.AccountAddress {
			v.Value.Enqueue(block)
			if v.Priority == nil {
				v.Priority = big.NewInt(0)
			}
			v.Priority.Add(v.Priority, block.Balance[Vite_TokenId])
			find = true
			break
		}
	}
	if !find {
		var bq BlockQueue
		bq.Enqueue(block)
		fItem := &FromItem{
			Key:      block.AccountAddress,
			Value:    &bq,
			Priority: block.Balance[Vite_TokenId],
		}
		pfq.Push(fItem)
	}
}

func (pfq *PriorityFromQueue) update(item *FromItem, key *types.Address, value *BlockQueue, priority *big.Int) {
	item.Value = value
	item.Priority = priority
	item.Key = key
	heap.Fix(pfq, item.Index)
}

func (pfq *PriorityFromQueue) Remove(key *types.Address) {
	for i := 0; i < pfq.Len(); i++ {
		if (*pfq)[i].Key == key {
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
	if (*ptq)[i].Priority.Cmp((*ptq)[j].Priority) == -1 {
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

func (ptq *PriorityToQueue) InsertNew(block *ledger.AccountBlock) {
	var find = false
	for _, v := range *ptq {
		if v.Key == block.ToAddress {
			v.Value.InsertNew(block)
			if v.Priority == nil {
				v.Priority = big.NewInt(0)
			}
			v.Priority.Add(v.Priority, block.Balance[Vite_TokenId])
			find = true
			break
		}
	}
	if !find {
		var fQueue PriorityFromQueue
		fQueue.InsertNew(block)
		tItem := &ToItem{
			Key:      block.ToAddress,
			Value:    &fQueue,
			Priority: block.Balance[Vite_TokenId],
		}
		ptq.Push(tItem)
	}
}

func (ptq *PriorityToQueue) update(item *ToItem, key *types.Address, value *PriorityFromQueue, priority *big.Int) {
	item.Value = value
	item.Priority = priority
	item.Key = key
	heap.Fix(ptq, item.Index)
}
