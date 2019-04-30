package net

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/vite/net/message"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/p2p"

	"github.com/vitelabs/go-vite/interfaces"

	"github.com/vitelabs/go-vite/log15"
)

// the minimal height difference between snapshot chain of ours and bestPeer,
// if the difference is little than this value, then we deem no need sync.
const minHeightDifference = 1000
const waitEnoughPeers = 10 * time.Second
const enoughPeers = 1
const chainGrowInterval = time.Second
const syncTaskSize = 3600

func shouldSync(from, to uint64) bool {
	if to >= from+minHeightDifference {
		return true
	}

	return false
}

func splitChunk(from, to uint64, size uint64) (chunks [][2]uint64) {
	// chunks may be only one block, then from == to
	if from > to || to == 0 {
		return
	}

	total := (to-from)/size + 1
	chunks = make([][2]uint64, total)

	var cTo uint64
	var i int
	for from <= to {
		if cTo = from + size - 1; cTo > to {
			cTo = to
		}

		chunks[i] = [2]uint64{from, cTo}

		from = cTo + 1
		i++
	}

	return chunks[:i]
}

type hashHeightNode struct {
	*ledger.HashHeight
	weight int
	nodes  map[types.Hash]*hashHeightNode
}

func newHashHeightTree() *hashHeightNode {
	return &hashHeightNode{
		nodes: make(map[types.Hash]*hashHeightNode),
	}
}

func (t *hashHeightNode) addBranch(list []*ledger.HashHeight) {
	var tree = t
	var subTree *hashHeightNode
	var ok bool
	for _, h := range list {
		subTree, ok = tree.nodes[h.Hash]
		if ok {
			subTree.weight++
		} else {
			subTree = &hashHeightNode{
				h,
				1,
				make(map[types.Hash]*hashHeightNode),
			}
			tree.nodes[h.Hash] = subTree
		}

		tree = subTree
	}
}

func (t *hashHeightNode) bestBranch() (list []*ledger.HashHeight) {
	var tree = t
	var subTree *hashHeightNode
	var weight int

	for {
		if len(tree.nodes) == 0 {
			return
		}

		weight = 0

		for _, n := range tree.nodes {
			if n.weight > weight {
				weight = n.weight
				subTree = n
			}
		}

		list = append(list, subTree.HashHeight)

		tree = subTree
	}
}

type syncPeerSet interface {
	sub(ch chan<- peerEvent)
	unSub(ch chan<- peerEvent)
	syncPeer() Peer
	bestPeer() Peer
}

type syncer struct {
	from, to uint64
	batchEnd uint64

	height uint64
	state  syncState

	pending int32 // atomic, pending for add tasks

	timeout time.Duration // sync error if timeout

	peers     syncPeerSet
	eventChan chan peerEvent // get peer add/delete event

	chain      syncChain // query current block and height
	downloader syncDownloader
	reader     syncCacheReader

	curSubId int // for subscribe
	subs     map[int]SyncStateCallback
	mu       sync.Mutex

	running int32
	term    chan struct{}
	log     log15.Logger
}

func (s *syncer) Peek() *Chunk {
	if s.reader == nil {
		return nil
	}
	return s.reader.Peek()
}

func (s *syncer) Pop(endHash types.Hash) {
	if s.reader != nil {
		s.reader.Pop(endHash)
	}
}

func (s *syncer) name() string {
	return "syncer"
}

func (s *syncer) codes() []code {
	return []code{CodeCheckResult, CodeHashList}
}

func (s *syncer) handle(msg p2p.Msg, sender Peer) (err error) {
	switch msg.Code {
	case CodeCheckResult:
		var hh = &ledger.HashHeight{}
		err = hh.Deserialize(msg.Payload)
		if err != nil {
			return
		}
		// todo

	case CodeHashList:
		var list = &message.HashHeightList{}
		err = list.Deserialize(msg.Payload)
		if err != nil {
			return
		}
		// todo

	}

	return nil
}

func newSyncer(chain syncChain, peers syncPeerSet, reader syncCacheReader, downloader syncDownloader, timeout time.Duration) *syncer {
	s := &syncer{
		chain:      chain,
		peers:      peers,
		eventChan:  make(chan peerEvent, 1),
		subs:       make(map[int]SyncStateCallback),
		downloader: downloader,
		reader:     reader,
		timeout:    timeout,
		log:        netLog.New("module", "syncer"),
	}

	s.state = syncStateInit{s}

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
	}
}

func (s *syncer) start() {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}

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

	syncPeer := s.peers.syncPeer()
	// all peers disconnected
	if syncPeer == nil {
		s.state.error(syncErrorNoPeers)
		s.log.Error("sync error: no peers")
		return
	}

	syncPeerHeight := syncPeer.height()

	// compare snapshot chain height and local cache height
	current := s.chain.GetLatestSnapshotBlock()
	// syncPeer is not all enough, no need to sync
	if !shouldSync(current.Height, syncPeerHeight) {
		s.log.Info(fmt.Sprintf("sync done: syncPeer %s at %d, our height: %d", syncPeer.String(), syncPeerHeight, current.Height))
		s.state.done()
		return
	}

	s.state.sync()
	s.sync(syncPeerHeight)

	// check chain height
	checkChainTicker := time.NewTicker(chainGrowInterval)
	defer checkChainTicker.Stop()
	var lastCheckTime = time.Now()

	for {
		select {
		case <-s.eventChan:
			// peers change, find new peer or peer disconnected. Choose sync peer again.
			if bestPeer := s.peers.bestPeer(); bestPeer != nil {
				if s.height >= bestPeer.height() {
					// we are taller than the best peer, no need sync
					s.log.Info(fmt.Sprintf("sync done: bestPeer %s at %d, our height: %d", bestPeer.String(), bestPeer.height(), s.height))
					s.state.done()

					// cancel rest tasks
					s.cancelTasks()
					s.reader.clean()
					return
				}

				syncPeer = s.peers.syncPeer()
				syncPeerHeight = syncPeer.height()
				if shouldSync(s.to, syncPeerHeight) {
					// only change sync target at two following scene:
					// 1. the bestPeer is low than current target, because taller peers maybe disconnected.
					// 2. the new syncPeer is taller enough than current target, because find more taller peers.
					s.setTarget(syncPeerHeight)
					s.state.sync()
				}
			} else {
				s.log.Error("sync error: no peers")
				s.state.error(syncErrorNoPeers)
				// no peers
				// cancel rest tasks, keep cache
				s.cancelTasks()
				s.reader.reset()
				return
			}

		case now := <-checkChainTicker.C:
			current = s.chain.GetLatestSnapshotBlock()

			if current.Height >= s.to {
				s.state.done()

				// clean cache, cancel rest tasks
				s.cancelTasks()
				s.reader.clean()
				return
			}

			if current.Height == s.height {
				if now.Sub(lastCheckTime) > s.timeout {
					s.log.Error("sync error: stuck")
					s.state.error(syncErrorStuck)
					// watching chain grow through fetcher
					// clean cache, cancel rest tasks
					s.cancelTasks()
					s.reader.clean()
				}
			} else {
				s.height = current.Height
				lastCheckTime = now
			}

		case <-s.term:
			s.state.cancel()

			// cancel rest tasks, keep cache
			s.cancelTasks()
			s.reader.reset()
			return
		}
	}
}

func (s *syncer) getLocalHeight() uint64 {
	current := s.chain.GetLatestSnapshotBlock().Height
	cacheHeight := s.reader.cacheHeight()

	if current < cacheHeight {
		current = cacheHeight
	}

	return current
}

func (s *syncer) sync(syncPeerHeight uint64) {
	local := s.getLocalHeight()
	s.from = local + 1
	s.to = syncPeerHeight

	if atomic.CompareAndSwapInt32(&s.pending, 0, 1) {
		go s.createTasks()
	}
}

func (s *syncer) cancelTasks() {
	local := s.getLocalHeight()

	s.downloader.cancel(local)
}

func (s *syncer) createTasks() {
	defer atomic.StoreInt32(&s.pending, 0)

	var from = s.from
	var to uint64

	for from < s.to {
		to = from + syncTaskSize - 1
		if to > s.to {
			to = s.to
		}

		if s.downloader.download(from, to, false) {
			from = to + 1
		} else {
			s.log.Warn(fmt.Sprintf("failed to download %d-%d", from, to))
			break
		}
	}
}

// this method will be called when our target Height changed, (eg. the best peer disconnected)
func (s *syncer) setTarget(to uint64) {
	if s.to > to {
		s.to = s.downloader.cancel(to + 1)
	} else {
		if atomic.LoadInt32(&s.pending) == 1 {
			s.to = to
		} else {
			s.sync(to)
		}
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
	From       uint64                 `json:"from"`
	To         uint64                 `json:"to"`
	Current    uint64                 `json:"current"`
	State      SyncState              `json:"state"`
	Downloader DownloaderStatus       `json:"downloader"`
	Cache      interfaces.SegmentList `json:"cache"`
}

func (s *syncer) Detail() SyncDetail {
	st := s.Status()

	return SyncDetail{
		From:       st.From,
		To:         st.To,
		Current:    st.Current,
		State:      st.State,
		Downloader: s.downloader.status(),
		Cache:      s.reader.chunks(),
	}
}
