package net

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

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
	SyncNotStart:   "Sync Not Start",
	Syncing:        "Synchronising",
	Syncdone:       "Sync done",
	Syncerr:        "Sync error",
	SyncCancel:     "Sync canceled",
	SyncDownloaded: "Sync all blocks Downloaded",
}

func (s SyncState) String() string {
	return syncStatus[s]
}

type SyncStateFeed struct {
	lock      sync.RWMutex
	currentId int
	subs      map[int]SyncStateCallback
}

func newSyncStateFeed() *SyncStateFeed {
	return &SyncStateFeed{
		subs: make(map[int]SyncStateCallback),
	}
}

func (s *SyncStateFeed) Sub(fn SyncStateCallback) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.currentId++
	s.subs[s.currentId] = fn
	return s.currentId
}

func (s *SyncStateFeed) Unsub(subId int) {
	if subId <= 0 {
		return
	}

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
// the minimal height difference between snapshot chain of ours and bestPeer
// if the difference is little than this value, then we deem no need sync
const minHeightDifference = 3600

var waitEnoughPeers = 10 * time.Second
var enoughPeers = 3

// todo should be set according to block number
var chainGrowTimeout = 5 * time.Minute
var downloadTimeout = 5 * time.Minute
var chainGrowInterval = 10 * time.Second

type syncer struct {
	from, to   uint64 // include
	count      uint64 // current amount of snapshotblocks have received
	total      uint64 // totol amount of snapshotblocks need download, equal: to - from + 1
	blocks     []*ledger.SnapshotBlock
	stLoc      sync.Mutex // protect: count blocks total
	state      SyncState
	term       chan struct{}
	downloaded chan struct{}
	feed       *SyncStateFeed
	chain      Chain // query latest block and genesis block
	peers      *peerSet
	pEvent     chan *peerEvent
	pool       RequestPool // add new request
	log        log15.Logger
	running    int32
	receiver   Receiver
	fc         *fileClient
	reqs       []*subLedgerRequest
}

func newSyncer(chain Chain, peers *peerSet, pool *requestPool, receiver Receiver, fc *fileClient) *syncer {
	s := &syncer{
		state:      SyncNotStart,
		term:       make(chan struct{}),
		downloaded: make(chan struct{}, 1),
		feed:       newSyncStateFeed(),
		chain:      chain,
		peers:      peers,
		pEvent:     make(chan *peerEvent),
		pool:       pool,
		log:        log15.New("module", "net/syncer"),
		receiver:   receiver,
		fc:         fc,
	}

	// subscribe peer add/del event
	peers.Sub(s.pEvent)

	return s
}

func (s *syncer) Stop() {
	select {
	case <-s.term:
	default:
		s.peers.Unsub(s.pEvent)
		close(s.term)
	}
}

func (s *syncer) Start() {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}

	defer atomic.StoreInt32(&s.running, 0)

	start := time.NewTimer(waitEnoughPeers)
	defer start.Stop()

	s.log.Info("start sync")

wait:
	for {
		select {
		case e := <-s.pEvent:
			if e.count >= enoughPeers {
				break wait
			}
		case <-start.C:
			break wait
		case <-s.term:
			s.setState(SyncCancel)
			return
		}
	}

	// for now syncState is SyncNotStart
	p := s.peers.BestPeer()
	if p == nil {
		s.setState(Syncerr)
		s.log.Error("sync error: no peers")
		return
	}

	// compare snapshot chain height
	current := s.chain.GetLatestSnapshotBlock()
	// p is lower than me, or p is not all enough
	if current.Height >= p.height || current.Height+minBlocks > p.height {
		if current.Height > p.height {
			p.SendNewSnapshotBlock(current)
		}
		s.log.Info(fmt.Sprintf("no need sync to bestPeer %s at %d, our height: %d", p, p.height, current.Height))
		s.setState(Syncdone)
		return
	}

	s.from = current.Height + 1
	s.to = p.height
	s.total = s.to - s.from + 1

	s.log.Info(fmt.Sprintf("syncing: current at %d, to %d", current.Height, s.to))
	s.setState(Syncing)

	// begin sync with peer
	s.blocks = make([]*ledger.SnapshotBlock, s.to-s.from+1)
	s.sync(s.from, s.to)

	// for now syncState is syncing
	deadline := time.NewTimer(downloadTimeout)
	defer deadline.Stop()
	// will setState follow
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case e := <-s.pEvent:
			if e.code == delPeer {
				// a taller peer is disconnected, maybe is the peer we need syncing with
				// because peer`s height is growing
				targetHeight := s.to
				if e.peer.height >= targetHeight {
					bestPeer := s.peers.BestPeer()
					if bestPeer != nil {
						if s.shouldSync(current.Height, bestPeer.height) {
							s.setTarget(bestPeer.height)
						} else {
							// no need sync
							s.log.Info(fmt.Sprintf("no need sync to bestPeer %s at %d, our height: %d", bestPeer, bestPeer.height, current.Height))
							s.setState(Syncdone)
							return
						}
					} else {
						// have no peers
						s.log.Error("sync error: no peers")
						s.setState(Syncerr)
						return
					}
				}
			}
		case <-s.downloaded:
			s.log.Info("sync downloaded")
			s.setState(SyncDownloaded)
			// check chain height timeout
			deadline.Reset(chainGrowTimeout)
			// check chain height loop
			ticker.Stop()
			ticker = time.NewTicker(chainGrowInterval)
		case <-deadline.C:
			s.log.Error("sync error: timeout")
			s.setState(Syncerr)
			return
		case <-ticker.C:
			current := s.chain.GetLatestSnapshotBlock()
			if current.Height >= s.to {
				s.log.Info(fmt.Sprintf("sync done, current height: %d", current.Height))
				s.setState(Syncdone)
				return
			}
			s.log.Info(fmt.Sprintf("current height: %d", current.Height))
		case <-s.term:
			s.log.Warn("sync cancel")
			s.setState(SyncCancel)
			return
		}
	}
}

func (s *syncer) shouldSync(from, to uint64) bool {
	if to > from {
		if to >= from+minHeightDifference {
			return true
		}
	}

	return false
}

// this method will be called when our target Height changed, (eg. the best peer disconnected)
func (s *syncer) setTarget(to uint64) {
	total2 := to - s.from + 1

	if to > s.to {
		record := make([]*ledger.SnapshotBlock, total2)
		copy(record, s.blocks)
		s.blocks = record

		// send taller task
		s.sync(s.to+1, to)
	} else {
		// update valid count
		for _, r := range s.blocks[total2:] {
			if r != nil {
				s.count--
			}
		}

		s.blocks = s.blocks[:total2]

		// todo cancel some taller task
	}

	s.total = total2
	s.to = to
}

func (s *syncer) sync(from, to uint64) {
	pieces := splitSubLedger(s.from, s.to, s.peers.Pick(s.from))

	for _, piece := range pieces {
		msgId := s.pool.MsgID()

		req := &subLedgerRequest{
			id:         msgId,
			from:       piece.from,
			to:         piece.to,
			peer:       piece.peer,
			expiration: time.Now().Add(10 * time.Minute),
			done:       s.reqCallback,
		}

		s.reqs = append(s.reqs, req)

		s.pool.Add(req)
	}
}

func (s *syncer) reqCallback(id uint64, err error) {
	if err != nil {
		s.setState(Syncerr)
	}
}

func (s *syncer) setState(t SyncState) {
	s.state = t
	s.feed.Notify(t)
}

func (s *syncer) SubscribeSyncStatus(fn SyncStateCallback) (subId int) {
	return s.feed.Sub(fn)
}

func (s *syncer) UnsubscribeSyncStatus(subId int) {
	s.feed.Unsub(subId)
}

func (s *syncer) offset(block *ledger.SnapshotBlock) uint64 {
	return block.Height - s.from
}

//func (s *syncer) insert(block *ledger.SnapshotBlock) {
//	offset := s.offset(block)
//
//	s.stLoc.Lock()
//	defer s.stLoc.Unlock()
//
//	if s.blocks[offset] == nil {
//		s.blocks[offset] = block
//		s.count++
//	}
//}

func (s *syncer) receiveBlocks(sblocks []*ledger.SnapshotBlock, ablocks []*ledger.AccountBlock) {
	s.receiver.ReceiveAccountBlocks(ablocks)
	s.receiver.ReceiveSnapshotBlocks(sblocks)

	count := atomic.AddUint64(&s.count, uint64(len(sblocks)))

	s.log.Info(fmt.Sprintf("receive %d snapshotblocks, %d accountblocks, process %d/%d", len(sblocks), len(ablocks), count, s.total))

	if atomic.LoadUint64(&s.count) >= s.total {
		// all blocks have downloaded
		s.downloaded <- struct{}{}
	}
}

type SyncStatus struct {
	From    uint64
	To      uint64
	Current uint64
	State   SyncState
}

func (s *syncer) Status() *SyncStatus {
	current := s.chain.GetLatestSnapshotBlock()

	return &SyncStatus{
		From:    s.from,
		To:      s.to,
		Current: current.Height,
		State:   s.state,
	}
}

func (s *syncer) SyncState() SyncState {
	return s.state
}
