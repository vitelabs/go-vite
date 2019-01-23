package net

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"

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

// the minimal height difference between snapshot chain of ours and bestPeer
// if the difference is little than this value, then we deem no need sync
const minHeightDifference = 3600
const waitEnoughPeers = 10 * time.Second
const enoughPeers = 3
const chainGrowInterval = time.Second

func shouldSync(from, to uint64) bool {
	if to >= from+minHeightDifference {
		return true
	}

	return false
}

type syncer struct {
	from, to uint64
	current  uint64 // current height
	aCount   uint64 // count of snapshot blocks have downloaded
	sCount   uint64 // count of account blocks have download

	state SyncState

	peers *peerSet

	pending int // pending count of FileList msg
	resChan chan *message.FileList

	// query current block and height
	chain Chain

	// get peer add/delete event
	eventChan chan peerEvent

	// handle blocks
	verifier Verifier
	notifier blockNotifier

	// for sync tasks
	fc   *fileClient
	pool *chunkPool
	exec syncTaskExecutor

	// for subscribe
	curSubId int
	subs     map[int]SyncStateCallback

	running int32
	term    chan struct{}
	log     log15.Logger
}

func (s *syncer) receiveAccountBlock(block *ledger.AccountBlock) error {
	err := s.verifier.VerifyNetAb(block)
	if err != nil {
		return err
	}

	s.notifier.notifyAccountBlock(block, types.RemoteSync)
	return nil
}

func (s *syncer) receiveSnapshotBlock(block *ledger.SnapshotBlock) error {
	err := s.verifier.VerifyNetSb(block)
	if err != nil {
		return err
	}

	s.notifier.notifySnapshotBlock(block, types.RemoteSync)
	return nil
}

func newSyncer(chain Chain, peers *peerSet, verifier Verifier, gid MsgIder, notifier blockNotifier) *syncer {
	s := &syncer{
		state:     SyncNotStart,
		chain:     chain,
		peers:     peers,
		eventChan: make(chan peerEvent, 1),
		verifier:  verifier,
		notifier:  notifier,
		subs:      make(map[int]SyncStateCallback),
		term:      make(chan struct{}),
		log:       log15.New("module", "net/syncer"),
	}

	pool := newChunkPool(peers, gid, s)
	fc := newFileClient(chain.Compressor(), s, peers)
	s.exec = newExecutor(s)

	s.pool = pool
	s.fc = fc

	return s
}

func (s *syncer) SubscribeSyncStatus(fn SyncStateCallback) int {
	s.curSubId++
	s.subs[s.curSubId] = fn
	return s.curSubId
}

func (s *syncer) UnsubscribeSyncStatus(subId int) {
	if subId <= 0 {
		return
	}

	delete(s.subs, subId)
}

func (s *syncer) SyncState() SyncState {
	return s.state
}

func (s *syncer) setState(st SyncState) {
	s.state = st
	for _, sub := range s.subs {
		sub(st)
	}
}

func (s *syncer) Stop() {
	select {
	case <-s.term:
	default:
		close(s.term)
		s.pool.stop()
		s.fc.stop()
	}
}

func (s *syncer) Start() {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}

	s.peers.Sub(s.eventChan)
	defer s.peers.UnSub(s.eventChan)

	start := time.NewTimer(waitEnoughPeers)

wait:
	for {
		select {
		case e := <-s.eventChan:
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
	syncPeer := s.peers.SyncPeer()
	if syncPeer == nil {
		s.setState(Syncerr)
		s.log.Error("sync error: no peers")
		return
	}

	syncPeerHeight := syncPeer.Height()

	// compare snapshot chain height
	current := s.chain.GetLatestSnapshotBlock()
	// p is not all enough, no need to sync
	if current.Height+minHeightDifference > syncPeerHeight {
		if current.Height < syncPeerHeight {
			syncPeer.Send(GetSnapshotBlocksCode, 0, &message.GetSnapshotBlocks{
				From:    ledger.HashHeight{Height: syncPeerHeight},
				Count:   1,
				Forward: true,
			})
		}

		s.log.Info(fmt.Sprintf("sync done: syncPeer %s at %d, our height: %d", syncPeer.RemoteAddr(), syncPeerHeight, current.Height))
		s.setState(Syncdone)
		return
	}

	s.current = current.Height
	s.from = current.Height + 1
	s.to = syncPeerHeight
	s.current = current.Height
	s.setState(Syncing)
	s.getSubLedgerFromAll()

	// check chain height
	checkChainTicker := time.NewTicker(chainGrowInterval)
	defer checkChainTicker.Stop()
	var lastChekTime time.Time

	for {
		select {
		case e := <-s.eventChan:
			if e.code == delPeer {
				// a taller peer is disconnected, maybe is the peer we syncing to
				// because peer`s height is growing
				if e.peer.Height() >= s.to {
					if syncPeer = s.peers.SyncPeer(); syncPeer != nil {
						syncPeerHeight = syncPeer.Height()
						if shouldSync(current.Height, syncPeerHeight) {
							s.setTarget(syncPeerHeight)
						} else {
							// no need sync
							s.log.Info(fmt.Sprintf("no need sync to bestPeer %s at %d, our height: %d", syncPeer, syncPeerHeight, current.Height))
							s.setState(Syncdone)
							return
						}
					} else {
						// have no peers
						s.log.Error("sync error: no peers")
						s.setState(Syncerr)
					}
				}
			} else if shouldSync(current.Height, e.peer.Height()) {
				s.getSubLedgerFrom(e.peer)
			}

		case now := <-checkChainTicker.C:
			current = s.chain.GetLatestSnapshotBlock()

			if current.Height >= s.to {
				s.log.Info(fmt.Sprintf("sync done, current height: %d", current.Height))
				s.setState(Syncdone)
				return
			}

			if current.Height == s.current && now.Sub(lastChekTime) > 10*time.Minute {
				s.setState(Syncerr)
			} else {
				s.exec.runTo(s.current + 3600)
			}

		case <-s.term:
			s.log.Warn("sync cancel")
			s.setState(SyncCancel)
			return
		}
	}
}

// this method will be called when our target Height changed, (eg. the best peer disconnected)
func (s *syncer) setTarget(to uint64) {
	atomic.StoreUint64(&s.to, to)
}

func (s *syncer) getSubLedgerFromAll() {
	l := s.peers.Pick(s.from + 1)

	for _, p := range l {
		s.getSubLedgerFrom(p)
	}
}

func (s *syncer) getSubLedgerFrom(p Peer) {
	from, to := s.from, s.to
	pTo := p.Height()
	if pTo > to {
		pTo = to
	}

	msg := &message.GetSubLedger{
		From:    ledger.HashHeight{Height: from},
		Count:   pTo - from + 1,
		Forward: true,
	}

	if err := p.Send(GetSubLedgerCode, 0, msg); err != nil {
		return
	} else {
		s.log.Info(fmt.Sprintf("get subledger from %d to %d to %s at %d", from, pTo, p.RemoteAddr(), p.Height()))
	}
}

func (s *syncer) pendingFileList() {
	var wait <-chan time.Time
	count := 0

	fileMap := make(map[filename]File)

Loop:
	for {
		select {
		case <-s.term:
			return
		case files := <-s.resChan:
			if wait == nil {
				wait = time.After(3 * time.Second)
			}

			for _, file := range files.Files {
				fileMap[file.Filename] = file
			}

			count++
			if count > 3 || count > s.pending/2 {
				break Loop
			}

		case <-wait:
			break Loop
		}
	}

	// prepare tasks
	if len(fileMap) > 0 {
		// has file
		files := make(Files, len(fileMap))
		i := 0
		for _, file := range fileMap {
			files[i] = file
			i++
		}

		sort.Sort(files)
		for _, file := range files {
			s.exec.add(&fileTask{
				file:       file,
				downloader: s.fc,
			})
		}
	} else {
		// use chunk
		cks := splitChunk(s.from, s.to, chunkSize)
		for _, ck := range cks {
			s.exec.add(&chunkTask{
				from:       ck[0],
				to:         ck[1],
				downloader: s.pool,
			})
		}
	}
}

func (s *syncer) ID() string {
	return "syncer"
}

func (s *syncer) Cmds() []ViteCmd {
	return []ViteCmd{FileListCode, SubLedgerCode}
}

func (s *syncer) Handle(msg *p2p.Msg, sender Peer) (err error) {
	switch ViteCmd(msg.Cmd) {
	case FileListCode:
		res := new(message.FileList)
		if err = res.Deserialize(msg.Payload); err != nil {
			return err
		}

		s.log.Info(fmt.Sprintf("receive %s from %s", res, sender.RemoteAddr()))

		if len(res.Files) > 0 {
			names := make([]filename, len(res.Files))
			for i, file := range res.Files {
				names[i] = file.Filename
			}
			s.fc.addFilePeer(names, sender)
		}

		s.resChan <- res

	case SubLedgerCode:
		return s.pool.Handle(msg, sender)
	}

	return nil
}

func (s *syncer) done(t syncTask) {
	from, to := t.bound()
	atomic.AddUint64(&s.sCount, to+1-from)
}

func (s *syncer) catch(t syncTask, err error) {
	if s.state != Syncing || atomic.LoadInt32(&s.running) == 0 {
		return
	}

	from, to := t.bound()
	target := atomic.LoadUint64(&s.to)

	if to <= target {
		s.setState(Syncerr)
		return
	}

	if from > s.to {
		return
	}

	s.pool.download(from, s.to)
}

func (s *syncer) nonTask(last syncTask) {
	_, to := last.bound()
	target := atomic.LoadUint64(&s.to)

	if to >= target {
		return
	}

	// use chunk
	cks := splitChunk(to+1, s.to, chunkSize)
	for _, ck := range cks {
		s.exec.add(&chunkTask{
			from:       ck[0],
			to:         ck[1],
			downloader: s.pool,
		})
	}
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
		Received: s.sCount,
		State:    s.state,
	}
}
