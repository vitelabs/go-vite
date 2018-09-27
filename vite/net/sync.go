package net

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var errSynced = errors.New("Syncing")

var waitEnoughPeers = 10 * time.Second
var enoughPeers = 3
var chainGrowTimeout = 5 * time.Minute
var downloadTimeout = 5 * time.Minute
var chainGrowInterval = time.Minute

type SyncState int32

const (
	SyncNotStart SyncState = iota
	Syncing
	Syncdone
	Syncerr
	SyncCancel
	SyncDownloaded
)

var syncStatus = [...]string{
	SyncNotStart: "Sync Not Start",
	Syncing:      "Synchronising",
	Syncdone:     "Sync done",
	Syncerr:      "Sync error",
	SyncCancel: "Sync canceled",
	SyncDownloaded: "Sync all blocks Downloaded",
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

// @section syncer
type syncer struct {
	from       *ledger.HashHeight	// exclude
	to     *ledger.HashHeight	// include
	state      SyncState // atomic
	term       chan struct{}
	downloaded chan struct{}
	feed       *SyncStateFeed
	chain      Chain
	peers      *peerSet
	pEvent chan *peerEvent
	pool *requestPool
	sblocks []*ledger.SnapshotBlock
	mblocks map[types.Address][]*ledger.AccountBlock
	sblockCount uint64 // atomic
	sblockCap  uint64
	log log15.Logger
}

func newSyncer(chain Chain, set *peerSet, pool *requestPool) *syncer {
	s := &syncer{
		state: SyncNotStart,
		term: make(chan struct{}),
		downloaded: make(chan struct{}, 1),
		chain: chain,
		peers: set,
		pEvent: make(chan *peerEvent),
		pool: pool,
		log: log15.New("module", "net/syncer"),
	}

	set.Sub(s.pEvent)

	return s
}

func (s *syncer) start() {
	start := time.NewTimer(waitEnoughPeers)
	defer start.Stop()

	wait:
	for {
		select {
		case e := <- s.pEvent:
			if e.count >= enoughPeers {
				break wait
			}
		case <- start.C:
			break wait
		case <- s.term:
			s.change(SyncCancel)
			return
		}
	}

// for now syncState is SyncNotStart
	p := s.peers.BestPeer()
	if p == nil {
		s.change(Syncerr)
		return
	}

	s.change(Syncing)

	p.SendMsg(&p2p.Msg{
		CmdSetID:   0,
		Cmd:        0,
		Id:         0,
		Size:       0,
		Payload:    nil,
	})

	// for now syncState is syncing
	deadline := time.NewTimer(downloadTimeout)
	defer deadline.Stop()
	// will change follow
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case e := <- s.pEvent:
			if e.code == delPeer {
				// a taller peer is disconnected
				targetHeight := s.to.Height
				if e.peer.height >= targetHeight {
					bestPeer := s.peers.BestPeer()
					if bestPeer != nil {
						// the best peer is lower than our targetHeight
						if bestPeer.height < targetHeight {
							s.setTarget(&ledger.HashHeight{
								Height:bestPeer.height,
								Hash: bestPeer.head,
							})
						} else if bestPeer.height < s.from.Height {
							// we are tallest
							s.change(Syncdone)
						}
					} else {
						// have no peers
						s.change(Syncerr)
					}
				}
			}
		case <- s.downloaded:
			s.change(SyncDownloaded)
			// check chain height timeout
			deadline.Reset(chainGrowTimeout)
			// check chain height loop
			ticker.Stop()
			ticker = time.NewTicker(chainGrowInterval)
		case <- deadline.C:
			s.change(Syncerr)
			return
		case <- ticker.C:
			current := s.chain.GetLatestSnapshotBlock()
			if current.Height >= s.to.Height {
				s.change(Syncdone)
				return
			}
		case <- s.term:
			s.change(SyncCancel)
			return
		}
	}
}

// this method will be called when our target Height changed, (eg. the best peer disconnected)
func (s *syncer) setTarget(to *ledger.HashHeight) {
	s.to = to
	recordCap := to.Height - s.from.Height

	if recordCap > s.sblockCap {
		record := make([]*ledger.SnapshotBlock, recordCap)
		copy(record, s.sblocks)
		s.sblocks = record
	} else {
		var cut uint64 = 0
		for _, r := range s.sblocks[recordCap:] {
			if r != nil {
				cut++
			}
		}

		s.sblocks = s.sblocks[:recordCap]
		s.sblockCount -= cut
	}

	s.sblockCap = recordCap

	// todo cancel some taller task
}

func (s *syncer) change(t SyncState) {
	s.state = t
	s.feed.Notify(t)
}

func (s *syncer) offset(block *ledger.SnapshotBlock) uint64 {
	return block.Height - s.from.Height - 1
}

func (s *syncer) insert(block *ledger.SnapshotBlock) bool {
	offset := s.offset(block)
	prev, current, next := s.sblocks[offset-1], s.sblocks[offset], s.sblocks[offset+1]

	if current == nil {
		// can insert
		if (prev == nil || block.PrevHash == prev.Hash) && (next == nil || next.PrevHash == block.Hash) {
			s.sblocks[offset] = block
			atomic.AddUint64(&s.sblockCount, 1)
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

			s.log.Error("fork", "prev", prevHash.String(), "current", block.Hash.String(), "nextPrev", nextPrevHash.String())
		}
	} else if current.Hash != block.Hash {
		// todo fork
		s.log.Error("fork", "current", current.Hash.String(), "new", block.Hash.String())
	}

	return false
}

func (s *syncer) receive(blocks []*ledger.SnapshotBlock, mblocks map[types.Address][]*ledger.AccountBlock) {
	for addr, ablocks := range mblocks {
		s.mblocks[addr] = append(s.mblocks[addr], ablocks...)
	}

	for _, block := range blocks {
		s.insert(block)
	}

	if atomic.LoadUint64(&s.sblockCount) == s.sblockCap {
		// all blocks have downloaded, then rank it
		for _, ablocks := range s.mblocks {
			accountblocks(ablocks).Sort()
		}
		snapshotblocks(s.sblocks).Sort()

		s.downloaded <- struct{}{}
	}
}

// @section helper
type accountblocks []*ledger.AccountBlock

func (a accountblocks) Len() int {
	return len(a)
}

func (a accountblocks) Less(i, j int) bool {
	return a[i].Height < a[j].Height
}

func (a accountblocks) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a accountblocks) Sort() {
	sort.Sort(a)
}

type snapshotblocks []*ledger.SnapshotBlock

func (a snapshotblocks) Len() int {
	return len(a)
}

func (a snapshotblocks) Less(i, j int) bool {
	return a[i].Height < a[j].Height
}

func (a snapshotblocks) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a snapshotblocks) Sort() {
	sort.Sort(a)
}

// @section MsgHandlers

