package net

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"sync"
	"time"
)

var errSynced = errors.New("Syncing")

var waitEnoughPeers = 10 * time.Second
var enoughPeers = 3
var waitForChainGrow = 5 * time.Minute

type SyncState int32

const (
	SyncNotStart SyncState = iota
	Syncing
	Syncdone
	Syncerr
	SyncCancel
)

var syncStatus = [...]string{
	SyncNotStart: "Sync Not Start",
	Syncing:      "Synchronising",
	Syncdone:     "Sync done",
	Syncerr:      "Sync error",
	SyncCancel: "Sync canceled",
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

type syncTask struct {
	from    uint64
	target  uint64
	state     SyncState // atomic
	downloaded    int32 // atomic, indicate whether the first sync data has download from bestPeer
	done chan struct{}
	newPeer chan struct{}
	feed     *SyncStateFeed
	chain BlockChain
	peers *peerSet
	peer *Peer
}

func (t *syncTask) start() {
	start := time.NewTimer(waitEnoughPeers)
	defer start.Stop()

	loop:
	for {
		select {
		case <- t.newPeer:
			if t.peers.Count() >= enoughPeers {
				break loop
			}
		case <- start.C:
			break loop
		}
	}

	p := t.peers.BestPeer()
	if p == nil {
		t.change(Syncerr)
		return
	}

	deadline := time.NewTimer(waitForChainGrow)
	defer deadline.Stop()
	p.SendMsg(&p2p.Msg{
		CmdSetID:   0,
		Cmd:        0,
		Id:         0,
		Size:       0,
		Payload:    nil,
	})
}

func (t *syncTask) change(s SyncState) {
	t.state = s
	t.feed.Notify(s)
}

type BlockChain interface {
	GetAccountBlockMap(AccountSegment) (map[string][]*ledger.AccountBlock, error)
	GetSnapshotBlocks(*Segment) ([]*ledger.SnapshotBlock, error)
	GetLatestAccountBlock(addr string) (*ledger.AccountBlock, error)
	GetLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetGenesesBlock() (*ledger.SnapshotBlock, error)
	GetSubLedger(startHeight uint64, endHeight uint64) ([]*ledger.SnapshotBlock, []*ledger.AccountBlock, error)
	GetAbHashList(segment *Segment) ([]types.Hash, error)
	GetSbHashList(segment *Segment) ([]types.Hash, error)
	GetSnapshotContent(snapshotBlockHash types.Hash)
}
