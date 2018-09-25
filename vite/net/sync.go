package net

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"sync"
	"sync/atomic"
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

// @section syncTask
type syncTask struct {
	from       *message.BlockID	// exclude
	to     *message.BlockID	// include
	state      SyncState // atomic
	downloaded int32 // atomic, indicate whether the first sync data has download from bestPeer
	done       chan struct{}
	newPeer    chan struct{}
	feed       *SyncStateFeed
	chain      Chain
	peers      *peerSet
	peer       *Peer
	record []*ledger.SnapshotBlock
	recordSize uint64 // atomic
	recordCap  uint64
	lock sync.Locker
	log log15.Logger
}

func newSyncTask(from, to *message.BlockID) *syncTask {
	recordCap := to.Height - from.Height

	return &syncTask{
		from: from,
		to: to,
		record: make([]*ledger.SnapshotBlock, recordCap),
		recordCap: recordCap,
		log: log15.New("module", "net/syncer"),
	}
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

// this method will be called when our target Height changed, (eg. the best peer disconnected)
func (t *syncTask) setTarget(to *message.BlockID) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.to = to
	recordCap := to.Height - t.from.Height

	if recordCap > t.recordCap {
		record := make([]*ledger.SnapshotBlock, recordCap)
		copy(record, t.record)
		t.record = record
	} else {
		var cut uint64 = 0
		for _, r := range t.record[recordCap:] {
			if r != nil {
				cut++
			}
		}

		t.record = t.record[:recordCap]
		t.recordSize -= cut
	}

	t.recordCap = recordCap
}

func (t *syncTask) change(s SyncState) {
	t.state = s
	t.feed.Notify(s)
}

func (t *syncTask) offset(block *ledger.SnapshotBlock) uint64 {
	return block.Height - t.from.Height - 1
}

func (t *syncTask) insert(block *ledger.SnapshotBlock) bool {
	offset := t.offset(block)
	prev, current, next := t.record[offset-1], t.record[offset], t.record[offset+1]

	if current == nil {
		// can insert
		if (prev == nil || block.PrevHash == prev.Hash) && (next == nil || next.PrevHash == block.Hash) {
			t.record[offset] = block
			atomic.AddUint64(&t.recordSize, 1)
			return true
		} else {
			// todo fork
			var prevHash, nextPrevHash types.Hash
			if prev != nil {
				prevHash = prev.Hash
			}
			if next != nil {
				nextPrevHash = next.PrevHash
			}

			t.log.Error("fork", "prev", prevHash.String(), "current", block.Hash.String(), "nextPrev", nextPrevHash.String())
		}
	} else if current.Hash != block.Hash {
		// todo fork
		t.log.Error("fork", "current", current.Hash.String(), "new", block.Hash.String())
	}

	return false
}

func (t *syncTask) receive(blocks []*ledger.SnapshotBlock) {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, block := range blocks {
		if t.insert(block) && atomic.LoadUint64(&t.recordSize) == t.recordCap {
			// all blocks have downloaded
			t.change(Syncdone)
			return
		}
	}
}
