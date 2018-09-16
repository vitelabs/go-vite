package net

import (
	"github.com/vitelabs/go-vite/ledger"
	"sync"
)

// @section snapshotblockfeed
type snapshotBlockFeed struct {
	lock      sync.RWMutex
	subs      map[int]func(*ledger.SnapshotBlock)
	currentId int
}

func (s *snapshotBlockFeed) Sub(fn func(*ledger.SnapshotBlock)) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.currentId++
	s.subs[s.currentId] = fn
	return s.currentId
}

func (s *snapshotBlockFeed) Unsub(subId int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.subs, subId)
}

func (s *snapshotBlockFeed) Notify(block *ledger.SnapshotBlock) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, fn := range s.subs {
		if fn != nil {
			go fn(block)
		}
	}
}

// @section accountBlockFeed
type accountBlockFeed struct {
	lock      sync.RWMutex
	subs      map[int]func(block *ledger.AccountBlock)
	currentId int
}

func (s *accountBlockFeed) Sub(fn func(*ledger.AccountBlock)) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.currentId++
	s.subs[s.currentId] = fn
	return s.currentId
}

func (s *accountBlockFeed) Unsub(subId int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.subs, subId)
}

func (s *accountBlockFeed) Notify(block *ledger.AccountBlock) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, fn := range s.subs {
		if fn != nil {
			go fn(block)
		}
	}
}
