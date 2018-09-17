package net

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
	"time"
)

var errSynced = errors.New("Syncing")

var waitEnoughPeers = 10 * time.Second
var enoughPeers = 3

type SyncState int

const (
	SyncNotStart SyncState = iota
	Syncing
	Syncdone
	Syncerr
)

var syncStatus = [...]string{
	SyncNotStart: "Sync Not Start",
	Syncing:      "Synchronising",
	Syncdone:     "Sync done",
	Syncerr:      "Sync error",
}

func (s SyncState) String() string {
	return syncStatus[s]
}

type SyncStateFeed struct {
	lock      sync.RWMutex
	currentId int
	subs      map[int]func(SyncState)
}

func (s *SyncStateFeed) Sub(fn func(SyncState)) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.currentId++
	s.subs[s.currentId] = fn
	return s.currentId
}

func (s *SyncStateFeed) Unsub(subId int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.subs, subId)
}
func (s *SyncStateFeed) Notify(st SyncState) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, fn := range s.subs {
		if fn != nil {
			go fn(st)
		}
	}
}

type BlockChain interface {
	GetAccountBlockMap(AccountSegment) (map[string][]*ledger.AccountBlock, error)
	GetSnapshotBlocks(*Segment) ([]*ledger.SnapshotBlock, error)
	GetLatestAccountBlock(addr string) (*ledger.AccountBlock, error)
	GetLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetGenesesBlock() (*ledger.SnapshotBlock, error)
	GetSubLedger(startHeight uint64, endHeight uint64) ([]*ledger.SnapshotBlock, []*ledger.AccountBlock, error)

	GetAbHashList(segment *Segment) ([]*types.Hash, error)
	GetSbHashList(segment *Segment) ([]*types.Hash, error)
	GetSnapshotContent(snapshotBlockHash *types.Hash)
	//GetSbAndSc(originBlockHash *types.Hash, count uint64, forward bool)([]*ledger.SnapshotBlock, []map, error)
}
