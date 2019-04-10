package net

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

// the minimal height difference between snapshot chain of ours and bestPeer,
// if the difference is little than this value, then we deem no need sync.
const minHeightDifference = 1800
const waitEnoughPeers = 10 * time.Second
const enoughPeers = 3
const chainGrowInterval = time.Second
const batchSyncTask = 24 // one day
const syncTaskSize = 3600

func shouldSync(from, to uint64) bool {
	if to >= from+minHeightDifference {
		return true
	}

	return false
}

func splitChunk(from, to uint64, chunk uint64) (chunks [][2]uint64) {
	// chunks may be only one block, then from == to
	if from > to || to == 0 {
		return
	}

	total := (to-from)/chunk + 1
	chunks = make([][2]uint64, total)

	var cTo uint64
	var i int
	for from <= to {
		if cTo = from + chunk - 1; cTo > to {
			cTo = to
		}

		chunks[i] = [2]uint64{from, cTo}

		from = cTo + 1
		i++
	}

	return chunks[:i]
}

type syncChain interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
}

type syncPeerSet interface {
	sub(ch chan<- peerEvent)
	unSub(ch chan<- peerEvent)
	syncPeer() Peer
}

type syncDownloader interface {
	stop()
	sync(from, to uint64)
	download(from, to uint64)
	subscribe(done func(err error))
	setTo(to uint64)
}

type syncer struct {
	from, to uint64
	height   uint64
	state    syncState

	timeout time.Duration

	peers     syncPeerSet
	eventChan chan peerEvent // get peer add/delete event

	chain      syncChain // query current block and height
	downloader syncDownloader

	curSubId int // for subscribe
	subs     map[int]SyncStateCallback
	mu       sync.Mutex

	running int32
	term    chan struct{}
	log     log15.Logger
}

func newSyncer(chain syncChain, peers syncPeerSet, downloader syncDownloader, timeout time.Duration) *syncer {
	s := &syncer{
		chain:      chain,
		peers:      peers,
		eventChan:  make(chan peerEvent, 1),
		subs:       make(map[int]SyncStateCallback),
		downloader: downloader,
		timeout:    timeout,
		log:        netLog.New("module", "syncer"),
	}

	s.state = syncStateInit{s}

	downloader.subscribe(s.done)

	return s
}

func (s *syncer) SubscribeSyncStatus(fn SyncStateCallback) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.curSubId++
	s.subs[s.curSubId] = fn
	return s.curSubId
}

func (s *syncer) UnsubscribeSyncStatus(subId int) {
	if subId <= 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subs, subId)
}

func (s *syncer) SyncState() SyncState {
	return s.state.state()
}

// implements syncStateHost
func (s *syncer) setState(state syncState) {
	s.state = state

	s.mu.Lock()
	subs := make([]SyncStateCallback, len(s.subs))
	i := 0
	for _, sub := range s.subs {
		subs[i] = sub
		i++
	}
	s.mu.Unlock()

	for _, sub := range subs {
		sub(state.state())
	}
}

func (s *syncer) stop() {
	if atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		s.peers.unSub(s.eventChan)
		close(s.term)
		s.downloader.stop()
	}
}

func (s *syncer) start() {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}

	s.state.wait()

	s.term = make(chan struct{})

	s.peers.sub(s.eventChan)

	defer s.stop()

	start := time.NewTimer(waitEnoughPeers)

Wait:
	for {
		select {
		case e := <-s.eventChan:
			if e.count >= enoughPeers {
				break Wait
			}
		case <-start.C:
			break Wait
		case <-s.term:
			s.state.cancel()
			start.Stop()
			return
		}
	}

	start.Stop()

	// for now syncState is SyncWait
	syncPeer := s.peers.syncPeer()
	if syncPeer == nil {
		s.state.error()
		s.log.Error("sync error: no peers")
		return
	}

	syncPeerHeight := syncPeer.height()

	// compare snapshot chain height
	current := s.chain.GetLatestSnapshotBlock()
	// syncPeer is not all enough, no need to sync
	if !shouldSync(current.Height, syncPeerHeight) {
		s.log.Info(fmt.Sprintf("sync done: syncPeer %s at %d, our height: %d", syncPeer.String(), syncPeerHeight, current.Height))
		s.state.done()
		return
	}

	s.height = current.Height
	s.from = s.height + 1
	s.to = syncPeerHeight
	s.state.start()
	s.downloader.sync(s.from, s.to)

	// check chain height
	checkChainTicker := time.NewTicker(chainGrowInterval)
	defer checkChainTicker.Stop()
	var lastCheckTime = time.Now()

	for {
		select {
		case <-s.eventChan:
			// choose sync peer again
			if syncPeer = s.peers.syncPeer(); syncPeer != nil {
				syncPeerHeight = syncPeer.height()
				if shouldSync(current.Height, syncPeerHeight) {
					s.setTarget(syncPeerHeight)
					s.state.start()
				} else {
					// no need sync
					s.log.Info(fmt.Sprintf("sync done: syncPeer %s at %d, our height: %d", syncPeer.String(), syncPeerHeight, current.Height))
					s.state.done()
					return
				}
			} else {
				s.log.Error("sync error: no peers")
				s.state.error()
				// no peers, then quit
				return
			}

		case now := <-checkChainTicker.C:
			current = s.chain.GetLatestSnapshotBlock()

			if current.Height >= s.to {
				s.state.done()
				return
			}

			if current.Height == s.height {
				if now.Sub(lastCheckTime) > s.timeout {
					s.log.Error("sync error: chain get stuck")
					s.state.error()
				}
			} else {
				s.height = current.Height
				lastCheckTime = now
			}

		case <-s.term:
			s.state.cancel()
			return
		}
	}
}

// this method will be called when our target Height changed, (eg. the best peer disconnected)
func (s *syncer) setTarget(to uint64) {
	atomic.StoreUint64(&s.to, to)
	s.downloader.setTo(to)
}

// subscribe sync downloader
func (s *syncer) done(err error) {
	if err != nil {
		s.state.error()
	} else {
		s.state.done()
	}
}

type SyncStatus struct {
	From    uint64
	To      uint64
	Current uint64
	State   SyncState
}

func (s *syncer) Status() SyncStatus {
	current := s.chain.GetLatestSnapshotBlock()

	return SyncStatus{
		From:    s.from,
		To:      s.to,
		Current: current.Height,
		State:   s.state.state(),
	}
}

type SyncDetail struct {
	From    uint64
	To      uint64
	Current uint64
	State   SyncState
	Tasks   []string
}

func (s *syncer) Detail() SyncDetail {
	st := s.Status()

	return SyncDetail{
		From:    st.From,
		To:      st.To,
		Current: st.Current,
		State:   st.State,
		Tasks:   nil,
	}
}
