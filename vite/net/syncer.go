package net

import (
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type SyncState uint

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
			fn := fn // closure
			common.Go(func() {
				fn(st)
			})
		}
	}
}

// @section syncer
// the minimal height difference between snapshot chain of ours and bestPeer
// if the difference is little than this value, then we deem no need sync
const minHeightDifference = 3600

var waitEnoughPeers = 10 * time.Second
var enoughPeers = 3
var chainGrowTimeout = 10 * time.Minute
var chainGrowInterval = 10 * time.Second

func shouldSync(from, to uint64) bool {
	if to >= from+minHeightDifference {
		return true
	}

	return false
}

type syncer struct {
	from, to uint64 // include
	count    uint64 // atomic, current amount of snapshotblocks have received
	total    uint64 // atomic, total amount of snapshotblocks need download, equal: to - from + 1
	//blocks     []uint64   // mark whether or not get the indexed block
	//sLock      sync.Mutex // protect blocks
	state      SyncState
	term       chan struct{}
	downloaded chan struct{}
	feed       *SyncStateFeed
	chain      Chain // query latest block and genesis block
	peers      *peerSet
	pEvent     chan *peerEvent
	pool       context // add new request
	log        log15.Logger
	running    int32
	receiver   Receiver
}

func newSyncer(chain Chain, peers *peerSet, pool context, receiver Receiver) *syncer {
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

	s.log.Info("prepare sync")

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
	// p is lower than me, or p is not all enough, no need to sync
	if current.Height >= p.height || current.Height+minSubLedger > p.height {
		// I`am not tall enough, then send my current block to p
		if current.Height > p.height && current.Height <= p.height+minSubLedger {
			p.SendNewSnapshotBlock(current)
		}

		s.log.Info(fmt.Sprintf("no need sync to bestPeer %s at %d, our height: %d", p, p.height, current.Height))
		s.setState(Syncdone)
		return
	}

	s.from = current.Height + 1
	s.to = p.height
	s.total = s.to - s.from + 1
	s.count = 0
	s.setState(Syncing)
	s.sync(s.from, s.to)
	s.log.Info(fmt.Sprintf("syncing: from %d, to %d", s.from, s.to))

	// check download timeout
	// check chain grow timeout
	checkTimer := time.NewTimer(u64ToDuration(s.total))
	defer checkTimer.Stop()

	// will be reset when downloaded
	checkChainTicker := time.NewTicker(24 * 365 * time.Hour)
	defer checkChainTicker.Stop()

	for {
		select {
		case e := <-s.pEvent:
			if e.code == delPeer {
				// a taller peer is disconnected, maybe is the peer we syncing to
				// because peer`s height is growing
				if e.peer.height >= s.to {
					if bestPeer := s.peers.BestPeer(); bestPeer != nil {
						if shouldSync(current.Height, bestPeer.height) {
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
			checkTimer.Reset(chainGrowTimeout)
			// check chain height loop
			checkChainTicker.Stop()
			checkChainTicker = time.NewTicker(chainGrowInterval)
		case <-checkTimer.C:
			s.log.Error("sync error: timeout")
			s.setState(Syncerr)
			return
		case <-checkChainTicker.C:
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

// this method will be called when our target Height changed, (eg. the best peer disconnected)
func (s *syncer) setTarget(to uint64) {
	if to == s.to {
		return
	}

	atomic.StoreUint64(&s.total, to-s.from+1)

	if to > s.to {
		s.sync(s.to+1, to)
	}

	s.to = to
}

func (s *syncer) counter(add bool, num uint64) {
	if num == 0 {
		return
	}

	var count uint64
	if add {
		count = atomic.AddUint64(&s.count, num)
	} else {
		count = atomic.AddUint64(&s.count, ^uint64(num-1))
	}

	// total maybe modified
	total := atomic.LoadUint64(&s.total)

	if s.state == SyncDownloaded {
		return
	}

	if count >= total {
		// all blocks have downloaded
		s.downloaded <- struct{}{}
	}
}

func (s *syncer) sync(from, to uint64) {
	pieces := splitSubLedger(from, to, s.peers.Pick(from+minSubLedger))

	for _, piece := range pieces {
		req := &subLedgerRequest{
			from:  piece.from,
			to:    piece.to,
			peer:  piece.peer,
			catch: s.reqError,
			rec:   s,
		}

		s.pool.Add(req)
	}
}

func (s *syncer) reqError(id uint64, err error) {
	if s.state != Syncing || atomic.LoadInt32(&s.running) != 1 {
		return
	}

	if r, ok := s.pool.Get(id); ok {
		from, to := r.Band()
		s.log.Error(fmt.Sprintf("GetSubLedger<%d-%d> error: %v", from, to, err))

		if from > s.to {
			return
		}

		if to > s.to {
			req := r.Req()
			req.SetBand(from, s.to)
			s.pool.Add(req)
		} else {
			s.setState(Syncerr)
		}
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

func (s *syncer) receiveSnapshotBlock(block *ledger.SnapshotBlock) {
	s.log.Info(fmt.Sprintf("syncer: receive SnapshotBlock %s/%d", block.Hash, block.Height))
	s.receiver.ReceiveSnapshotBlock(block)
	s.counter(true, 1)
}

func (s *syncer) receiveAccountBlock(block *ledger.AccountBlock) {
	s.log.Info(fmt.Sprintf("syncer: receive AccountBlock %s/%d", block.Hash, block.Height))
	s.receiver.ReceiveAccountBlock(block)
}

type SyncStatus struct {
	From     uint64
	To       uint64
	Current  uint64
	Received uint64
	State    SyncState
}

func (s *syncer) Status() *SyncStatus {
	current := s.chain.GetLatestSnapshotBlock()

	return &SyncStatus{
		From:     s.from,
		To:       s.to,
		Current:  current.Height,
		Received: s.count,
		State:    s.state,
	}
}

func (s *syncer) SyncState() SyncState {
	return s.state
}
