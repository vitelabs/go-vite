package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type blockNotifier struct {
	aSubs       map[int]AccountblockCallback
	bSubs       map[int]SnapshotBlockCallback
	currentId   int
	knownBlocks blockFilter
}

func (bf *blockNotifier) SubscribeAccountBlock(fn AccountblockCallback) (subId int) {
	bf.currentId++
	bf.aSubs[bf.currentId] = fn
	return bf.currentId
}

func (bf *blockNotifier) UnsubscribeAccountBlock(subId int) {
	delete(bf.aSubs, subId)
}

func (bf *blockNotifier) SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int) {
	bf.currentId++
	bf.bSubs[bf.currentId] = fn
	return bf.currentId
}

func (bf *blockNotifier) UnsubscribeSnapshotBlock(subId int) {
	delete(bf.aSubs, subId)
}

func (bf *blockNotifier) NotifySnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource) {
	ok := bf.knownBlocks.lookAndRecord(block.Hash[:])

	if ok {
		return
	}

	for _, fn := range bf.bSubs {
		if fn != nil {
			fn(block, source)
		}
	}
}

func (bf *blockNotifier) NotifyAccountBlock(block *ledger.AccountBlock, source types.BlockSource) {
	ok := bf.knownBlocks.lookAndRecord(block.Hash[:])

	if ok {
		return
	}

	for _, fn := range bf.aSubs {
		if fn != nil {
			fn(block.AccountAddress, block, source)
		}
	}
}
