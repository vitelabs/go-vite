package net

import (
	"fmt"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
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
	if s > SyncDownloaded {
		return "unknown sync state"
	}
	return syncStatus[s]
}

type SyncStateFeed struct {
	currentId int
	subs      map[int]SyncStateCallback
}

func newSyncStateFeed() *SyncStateFeed {
	return &SyncStateFeed{
		subs: make(map[int]SyncStateCallback),
	}
}

func (s *SyncStateFeed) Sub(fn SyncStateCallback) int {
	s.currentId++
	s.subs[s.currentId] = fn
	return s.currentId
}

func (s *SyncStateFeed) Unsub(subId int) {
	if subId <= 0 {
		return
	}

	delete(s.subs, subId)
}

func (s *SyncStateFeed) Notify(st SyncState) {
	for _, fn := range s.subs {
		if fn != nil {
			fn(st)
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

type fileSrc struct {
	peers
	state reqState
}

type pFileRecord struct {
	files
	index int
}

type syncer struct {
	from, to   uint64 // include
	count      uint64 // atomic, current amount of snapshotblocks have received
	total      uint64 // atomic, total amount of snapshotblocks need download, equal: to - from + 1
	peers      *peerSet
	state      SyncState
	downloaded chan struct{}
	feed       *SyncStateFeed
	chain      Chain // query latest block
	pEvent     chan *peerEvent
	receiver   Receiver
	fc         *fileClient
	pool       *chunkPool
	chunked    int32
	running    int32
	term       chan struct{}
	log        log15.Logger
}

func newSyncer(chain Chain, peers *peerSet, gid MsgIder, receiver Receiver) *syncer {
	s := &syncer{
		state:      SyncNotStart,
		term:       make(chan struct{}),
		downloaded: make(chan struct{}, 1),
		feed:       newSyncStateFeed(),
		chain:      chain,
		peers:      peers,
		pEvent:     make(chan *peerEvent, 1),
		log:        log15.New("module", "net/syncer"),
		receiver:   receiver,
	}

	// subscribe peer add/del event
	peers.Sub(s.pEvent)

	pool := newChunkPool(peers, gid, s)
	fc := newFileClient(chain, pool, s)

	s.pool = pool
	s.fc = fc

	return s
}

func (s *syncer) Stop() {
	select {
	case <-s.term:
	default:
		s.peers.UnSub(s.pEvent)
		close(s.term)
		s.pool.stop()
		s.fc.stop()
	}
}

func (s *syncer) Start() {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}

	defer atomic.StoreInt32(&s.running, 0)
	defer atomic.StoreInt32(&s.chunked, 0)

	s.pool.start()
	s.fc.start()

	defer s.pool.stop()
	defer s.fc.stop()

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
	// p is not all enough, no need to sync
	if current.Height+minSubLedger > p.height {
		if current.Height < p.height {
			p.Send(GetSnapshotBlocksCode, 0, &message.GetSnapshotBlocks{
				From:    ledger.HashHeight{Hash: p.head},
				Count:   1,
				Forward: true,
			})
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

	s.log.Info(fmt.Sprintf("syncing: from %d, to %d, bestPeer %s", s.from, s.to, p.RemoteAddr()))

	// check download timeout
	// check chain grow timeout
	checkTimer := time.NewTimer(u64ToDuration(s.total * 1000))
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
			s.log.Debug(fmt.Sprintf("current height: %d", current.Height))
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
	s.log.Debug(fmt.Sprintf("syncer: from %d to %d", from, to))

	peerList := s.peers.Pick(from)

	msg := &message.GetSubLedger{
		From:    ledger.HashHeight{Height: from},
		Count:   to - from + 1,
		Forward: true,
	}

	for _, peer := range peerList {
		peer.Send(GetSubLedgerCode, 0, msg)
	}
}

func (s *syncer) ID() string {
	return "syncer"
}

func (s *syncer) Cmds() []ViteCmd {
	return []ViteCmd{FileListCode, SubLedgerCode, ExceptionCode}
}

func (s *syncer) Handle(msg *p2p.Msg, sender Peer) error {
	cmd := ViteCmd(msg.Cmd)
	if cmd == FileListCode {
		res := new(message.FileList)

		if err := res.Deserialize(msg.Payload); err != nil {
			s.log.Error(fmt.Sprintf("descerialize %s from %s error: %v", res, sender.RemoteAddr(), err))
			return err
		}

		s.log.Info(fmt.Sprintf("receive %s from %s", res, sender.RemoteAddr()))

		if len(res.Files) > 0 {
			s.fc.gotFiles(res.Files, sender)
		}

		if sender.Height() >= s.to && len(res.Chunks) > 0 {
			if atomic.CompareAndSwapInt32(&s.chunked, 0, 1) {
				for _, c := range res.Chunks {
					if c[1] > 0 {
						// split to small chunks
						cs := splitChunk(c[0], c[1])
						for _, chunk := range cs {
							s.pool.add(&chunkRequest{from: chunk[0], to: chunk[1]})
						}
					}
				}
			}
		}
	} else if cmd == SubLedgerCode {
		s.pool.Handle(msg, sender)
	} else {
		netLog.Error(fmt.Sprintf("getSubLedgerHandler got %d need %d", msg.Cmd, SubLedgerCode))
	}

	return nil
}

func (s *syncer) catch(c piece) {
	if s.state != Syncing || atomic.LoadInt32(&s.running) == 0 {
		return
	}

	from, to := c.band()

	if from > s.to {
		return
	}

	if to > s.to {
		s.pool.add(&chunkRequest{from: from, to: s.to})
		s.log.Warn(fmt.Sprintf("retry request<%d-%d>", from, s.to))
	} else {
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

func (s *syncer) receiveSnapshotBlock(block *ledger.SnapshotBlock) {
	s.log.Debug(fmt.Sprintf("syncer: receive SnapshotBlock %s/%d", block.Hash, block.Height))
	s.receiver.ReceiveSnapshotBlock(block)
	s.counter(true, 1)
}

func (s *syncer) receiveAccountBlock(block *ledger.AccountBlock) {
	s.log.Debug(fmt.Sprintf("syncer: receive AccountBlock %s/%d", block.Hash, block.Height))
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
