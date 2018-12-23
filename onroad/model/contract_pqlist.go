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

func (q *contractBlocksPQueue) Push(x interface{}) {
	item := x.(*contractBlock)
	item.Index = len(*q)
	*q = append(*q, item)
}

func (q *contractBlocksPQueue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*q = old[0 : n-1]
	return item
}

func (q *contractBlocksPQueue) Len() int { return len(*q) }

func (q *contractBlocksPQueue) Less(i, j int) bool {
	return (*q)[i].Height < (*q)[j].Height
}

func (q *contractBlocksPQueue) Swap(i, j int) {
	(*q)[i], (*q)[j] = (*q)[j], (*q)[i]
	(*q)[i].Index = i
	(*q)[j].Index = j
}

type contractCallerBlocks struct {
	addr   types.Address
	blocks contractBlocksPQueue
}

type ContractCallerList struct {
	list         []contractCallerBlocks
	listMutex    sync.RWMutex
	currentIndex int

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
	for {
		if cList.list[cList.currentIndex].blocks.Len() > 0 {
			break
		}
		cList.currentIndex = (cList.currentIndex + 1) % len(cList.list)
	}
	if cb, ok := heap.Pop(&cList.list[cList.currentIndex].blocks).(*contractBlock); ok && cb != nil {
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
		block:  *block,
		Height: block.Height,
	}
	cList.listMutex.Lock()
	defer cList.listMutex.Unlock()
	var isAddressNew = true
	for _, v := range cList.list {
		if v.addr == block.AccountAddress {
			heap.Push(&v.blocks, cb)
			isAddressNew = false
		}
	}
	if isAddressNew == true {
		callerBlocks := &contractCallerBlocks{
			addr:   block.AccountAddress,
			blocks: make(contractBlocksPQueue, 0),
		}
		heap.Push(&callerBlocks.blocks, cb)
		cList.list = append(cList.list, *callerBlocks)
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
	cList.currentIndex = -1
}
