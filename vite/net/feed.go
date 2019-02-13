package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

// @section snapshotblockfeed
type snapshotBlockFeed struct {
	subs      map[int]SnapshotBlockCallback
	currentId int
}

func newSnapshotBlockFeed() *snapshotBlockFeed {
	return &snapshotBlockFeed{
		subs: make(map[int]SnapshotBlockCallback),
	}
}

func (s *snapshotBlockFeed) Sub(fn SnapshotBlockCallback) int {
	s.currentId++
	s.subs[s.currentId] = fn
	return s.currentId
}

func (s *snapshotBlockFeed) Unsub(subId int) {
	if subId <= 0 {
		return
	}

	delete(s.subs, subId)
}

func (s *snapshotBlockFeed) Notify(block *ledger.SnapshotBlock, source types.BlockSource) {
	for _, fn := range s.subs {
		if fn != nil {
			fn(block, source)
		}
	}
}

// @section accountBlockFeed
type accountBlockFeed struct {
	subs      map[int]AccountblockCallback
	currentId int
}

func newAccountBlockFeed() *accountBlockFeed {
	return &accountBlockFeed{
		subs: make(map[int]AccountblockCallback),
	}
}

func (s *accountBlockFeed) Sub(fn AccountblockCallback) int {
	s.currentId++
	s.subs[s.currentId] = fn
	return s.currentId
}

func (s *accountBlockFeed) Unsub(subId int) {
	if subId <= 0 {
		return
	}

	delete(s.subs, subId)
}

func (s *accountBlockFeed) Notify(block *ledger.AccountBlock, source types.BlockSource) {
	for _, fn := range s.subs {
		if fn != nil {
			fn(block.AccountAddress, block, source)
		}
	}
}
