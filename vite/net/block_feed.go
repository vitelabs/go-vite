package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type blockFeeder interface {
	BlockSubscriber
	blockNotifier
}

type blockNotifier interface {
	notifySnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource)
	notifyAccountBlock(block *ledger.AccountBlock, source types.BlockSource)
}

type chunkNotifier interface {
	notifyChunks(chunks []ledger.SnapshotChunk, source types.BlockSource)
}

type blockReceiver interface {
	receiveAccountBlock(block *ledger.AccountBlock, source types.BlockSource) error
	receiveSnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource) error
}

type blockFeed struct {
	aSubs     map[int]AccountBlockCallback
	bSubs     map[int]SnapshotBlockCallback
	currentId int
}

func newBlockFeeder() blockFeeder {
	return &blockFeed{
		aSubs: make(map[int]AccountBlockCallback),
		bSubs: make(map[int]SnapshotBlockCallback),
	}
}

func (bf *blockFeed) SubscribeAccountBlock(fn AccountBlockCallback) (subId int) {
	bf.currentId++
	bf.aSubs[bf.currentId] = fn
	return bf.currentId
}

func (bf *blockFeed) UnsubscribeAccountBlock(subId int) {
	delete(bf.aSubs, subId)
}

func (bf *blockFeed) SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int) {
	bf.currentId++
	bf.bSubs[bf.currentId] = fn
	return bf.currentId
}

func (bf *blockFeed) UnsubscribeSnapshotBlock(subId int) {
	delete(bf.aSubs, subId)
}

func (bf *blockFeed) notifySnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource) {
	for _, fn := range bf.bSubs {
		if fn != nil {
			fn(block, source)
		}
	}
}

func (bf *blockFeed) notifyAccountBlock(block *ledger.AccountBlock, source types.BlockSource) {
	for _, fn := range bf.aSubs {
		if fn != nil {
			fn(block.AccountAddress, block, source)
		}
	}
}

type safeBlockNotifier struct {
	blockFeeder
	Verifier
}

func (s *safeBlockNotifier) receiveAccountBlock(block *ledger.AccountBlock, source types.BlockSource) error {
	err := s.Verifier.VerifyNetAb(block)
	if err != nil {
		return err
	}

	s.blockFeeder.notifyAccountBlock(block, source)
	return nil
}

func (s *safeBlockNotifier) receiveSnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource) error {
	err := s.Verifier.VerifyNetSb(block)
	if err != nil {
		return err
	}

	s.blockFeeder.notifySnapshotBlock(block, source)
	return nil
}
