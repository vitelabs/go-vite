package model

import (
	"container/heap"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
)

type contractBlock struct {
	block  ledger.AccountBlock
	Index  int
	Height uint64
}

type contractBlocksPQueue []*contractBlock

func (q contractBlocksPQueue) Len() int { return len(q) }

func (q contractBlocksPQueue) Less(i, j int) bool {
	return q[i].Height < q[j].Height
}

func (q contractBlocksPQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].Index = i
	q[j].Index = j
}

func (q *contractBlocksPQueue) Push(x interface{}) {
	item := x.(*contractBlock)
	item.Index = len(*q)
	*q = append(*q, item)
}

func (q *contractBlocksPQueue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	//fmt.Printf("\n----Pop item (%v,%v)", item.block.Height, item.Index)
	item.Index = -1 // for safety
	*q = old[0 : n-1]
	return item
}

type contractCallerBlocks struct {
	addr   types.Address
	bQueue contractBlocksPQueue
}

type ContractCallerList struct {
	list         []*contractCallerBlocks
	currentIndex int
	listMutex    sync.RWMutex

	totalNum int
	numMutex sync.Mutex
}

func (cList *ContractCallerList) Len() int { return len(cList.list) }

func (cList *ContractCallerList) TxRemain() int {
	cList.numMutex.Lock()
	defer cList.numMutex.Unlock()
	return cList.totalNum
}

func (cList *ContractCallerList) GetNextTx() *ledger.AccountBlock {
	if cList.TxRemain() <= 0 {
		return nil
	}
	cList.listMutex.Lock()
	defer cList.listMutex.Unlock()
	for i := 1; i <= cList.Len(); i++ {
		if cList.list[cList.currentIndex].bQueue.Len() > 0 {
			break
		}
		cList.currentIndex = (cList.currentIndex + i) % len(cList.list)
	}
	if cb, ok := heap.Pop(&cList.list[cList.currentIndex].bQueue).(*contractBlock); ok && cb != nil {
		cList.SubTotalNum()
		return &cb.block
	}
	return nil
}

func (cList *ContractCallerList) GetCurrentIndex() int {
	cList.listMutex.Lock()
	defer cList.listMutex.Unlock()
	return cList.currentIndex
}

func (cList *ContractCallerList) AddNewTx(block *ledger.AccountBlock) {
	if block == nil {
		return
	}
	cb := &contractBlock{
		Height: block.Height,
		block:  *block,
	}
	cList.listMutex.Lock()
	defer cList.listMutex.Unlock()
	var isAddressNew = true
	for _, v := range cList.list {
		if v.addr == block.AccountAddress {
			heap.Push(&v.bQueue, cb)
			isAddressNew = false
		}
	}
	if isAddressNew == true {
		var q contractBlocksPQueue
		heap.Push(&q, cb)
		cList.list = append(cList.list, &contractCallerBlocks{
			addr:   block.AccountAddress,
			bQueue: q})
	}
	cList.AddTotalNum()
}

func (cList *ContractCallerList) AddTotalNum() {
	cList.numMutex.Lock()
	defer cList.numMutex.Unlock()
	cList.totalNum++
}

func (cList *ContractCallerList) SubTotalNum() {
	cList.numMutex.Lock()
	defer cList.numMutex.Unlock()
	cList.totalNum--
}

func (cList *ContractCallerList) removeCaller() {
	cList.listMutex.Lock()
	defer cList.listMutex.Unlock()
	if cList.Len() >= 0 {
		i := cList.currentIndex
		cList.list = append(cList.list[:i], cList.list[i+1:]...)
	}
	cList.currentIndex = 0
}
