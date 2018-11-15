package net

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
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
var chainGrowInterval = time.Second

func shouldSync(from, to uint64) bool {
	if to >= from+minHeightDifference {
		return true
	}

	return false
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

	// prepare to request file
	s.fc.start()
	defer s.fc.stop()
	// stop chunk pool
	defer s.pool.stop()

	start := time.NewTimer(waitEnoughPeers)

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
			s.log.Warn("sync cancel")
			s.setState(SyncCancel)
			start.Stop()
			return
		}
	}

	start.Stop()

	// for now syncState is SyncNotStart
	p := s.peers.SyncPeer()
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

		s.log.Info(fmt.Sprintf("sync done: bestPeer %s at %d, our height: %d", p.RemoteAddr(), p.height, current.Height))
		s.setState(Syncdone)
		return
	}

	s.from = current.Height + 1
	s.to = p.height
	s.total = s.to - s.from + 1
	s.count = 0
	s.setState(Syncing)
	s.sync()

	// check chain grow timeout
	var timeoutChan <-chan time.Time

	// check chain height
	checkChainTicker := time.NewTicker(chainGrowInterval)
	defer checkChainTicker.Stop()

	var speed uint64 = 100
	prevHeight := current.Height

	for {
		select {
		case e := <-s.pEvent:
			if e.code == delPeer {
				// a taller peer is disconnected, maybe is the peer we syncing to
				// because peer`s height is growing
				if e.peer.height >= s.to {
					if targetPeer := s.peers.SyncPeer(); targetPeer != nil {
						if shouldSync(current.Height, targetPeer.height) {
							s.setTarget(targetPeer.height)
						} else {
							// no need sync
							s.log.Info(fmt.Sprintf("no need sync to bestPeer %s at %d, our height: %d", targetPeer, targetPeer.height, current.Height))
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
			timeoutChan = time.NewTimer(u64ToDuration(s.total * 1000)).C

		case <-timeoutChan:
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

			speed = speed/2 + (current.Height-prevHeight)/2
			if speed == 0 {
				speed = 100
			} else if speed > 200 {
				speed = 200
			}

			s.fc.threshold(current.Height + 30*speed)
			s.pool.threshold(current.Height + 30*speed)
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
	if to == atomic.LoadUint64(&s.to) {
		return
	}

	atomic.StoreUint64(&s.total, to-s.from+1)
	atomic.StoreUint64(&s.to, to)

	if s.count >= s.total {
		select {
		case s.downloaded <- struct{}{}:
		default:
			// nothing
		}
	}
}

func (s *syncer) inc() {
	if s.state == SyncDownloaded {
		return
	}

	count := atomic.AddUint64(&s.count, 1)

	if count >= s.total {
		// all blocks have downloaded
		s.downloaded <- struct{}{}
	}
}

func (s *syncer) sync() {
	peerList := s.peers.Pick(s.from + 1)

	var msg *message.GetSubLedger

	from, to := s.from, s.to
	var pTo, pHeight uint64
	for _, peer := range peerList {
		pHeight = peer.Height()
		if pHeight > to {
			pTo = to
		} else {
			pTo = pHeight
		}

		msg = &message.GetSubLedger{
			From:    ledger.HashHeight{Height: from},
			Count:   pTo - from + 1,
			Forward: true,
		}

		peer.Send(GetSubLedgerCode, 0, msg)

		s.log.Info(fmt.Sprintf("sync from %d to %d to %s at %d", from, pTo, peer.RemoteAddr(), peer.Height()))
	}
}

func (s *syncer) ID() string {
	return "syncer"
}

func (s *syncer) Cmds() []ViteCmd {
	return []ViteCmd{FileListCode, SubLedgerCode}
}

func (s *syncer) Handle(msg *p2p.Msg, sender Peer) error {
	if ViteCmd(msg.Cmd) == FileListCode {
		res := new(message.FileList)

		if err := res.Deserialize(msg.Payload); err != nil {
			s.log.Error(fmt.Sprintf("descerialize %s from %s error: %v", res, sender.RemoteAddr(), err))
			return err
		}

		s.log.Info(fmt.Sprintf("receive %s from %s", res, sender.RemoteAddr()))

		if len(res.Files) > 0 {
			s.fc.gotFiles(res.Files, sender)
		}
	} else {
		s.pool.Handle(msg, sender)
	}

	return nil
}

func (s *syncer) catch(c piece) {
	if s.state != Syncing || atomic.LoadInt32(&s.running) == 0 {
		return
	}

	// no peers
	if bestPeer := s.peers.BestPeer(); bestPeer == nil {
		s.setState(Syncerr)
	} else if atomic.LoadUint64(&s.to) > bestPeer.Height() {
		// our target is taller than bestPeer, maybe bestPeer fallback
		s.setTarget(bestPeer.Height())
	}

	from, to := c.band()
	// piece is too taller, out of our sync target
	if from > s.to {
		return
	}

	newTo := to
	if newTo > s.to {
		newTo = s.to
	}

	if from > newTo {
		return
	}

	s.pool.add(from, s.to)
	s.log.Warn(fmt.Sprintf("retry sync from %d to %d", from, s.to))
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

//func (s *syncer) offset(block *ledger.SnapshotBlock) uint64 {
//	return block.Height - s.from
//}

func (s *syncer) receiveSnapshotBlock(block *ledger.SnapshotBlock) {
	s.log.Debug(fmt.Sprintf("syncer: receive SnapshotBlock %s/%d", block.Hash, block.Height))
	s.receiver.ReceiveSnapshotBlock(block)
	s.inc()
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
