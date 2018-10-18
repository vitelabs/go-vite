package net

import (
	"github.com/vitelabs/go-vite/ledger"
)

// @section snapshotblockfeed
type snapshotBlockFeed struct {
	//lock      sync.RWMutex
	subs      map[int]SnapshotBlockCallback
	currentId int
}

func newSnapshotBlockFeed() *snapshotBlockFeed {
	return &snapshotBlockFeed{
		subs: make(map[int]SnapshotBlockCallback),
	}
}

func (s *snapshotBlockFeed) Sub(fn SnapshotBlockCallback) int {
	//s.lock.Lock()
	//defer s.lock.Unlock()

	s.currentId++
	s.subs[s.currentId] = fn
	return s.currentId
}

func (s *snapshotBlockFeed) Unsub(subId int) {
	if subId <= 0 {
		return
	}

	//s.lock.Lock()
	//defer s.lock.Unlock()

	delete(s.subs, subId)
}

func (s *snapshotBlockFeed) Notify(block *ledger.SnapshotBlock) {
	//s.lock.RLock()
	//defer s.lock.RUnlock()
	for _, fn := range s.subs {
		if fn != nil {
			fn(block)
		}
	}
}

// @section accountBlockFeed
type accountBlockFeed struct {
	//lock      sync.RWMutex
	subs      map[int]AccountblockCallback
	currentId int
}

func newAccountBlockFeed() *accountBlockFeed {
	return &accountBlockFeed{
		subs: make(map[int]AccountblockCallback),
	}
}

func (s *accountBlockFeed) Sub(fn AccountblockCallback) int {
	//s.lock.Lock()
	//defer s.lock.Unlock()

	s.currentId++
	s.subs[s.currentId] = fn
	return s.currentId
}

func (s *accountBlockFeed) Unsub(subId int) {
	if subId <= 0 {
		return
	}

	//s.lock.Lock()
	//defer s.lock.Unlock()

	delete(s.subs, subId)
}

func (s *accountBlockFeed) Notify(block *ledger.AccountBlock) {
	//s.lock.RLock()
	//defer s.lock.RUnlock()
	for _, fn := range s.subs {
		if fn != nil {
			fn(block.AccountAddress, block)
		}
	}
}
