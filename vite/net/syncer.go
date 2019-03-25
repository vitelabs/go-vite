package net

import (
	"fmt"
	"sort"
	"sync"
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
	SyncDownloaded: "Sync downloaded",
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

type fileRecord struct {
	File
	add bool
}

type syncer struct {
	from, to uint64
	current  uint64 // current height
	aCount   uint64 // count of snapshot blocks have downloaded
	sCount   uint64 // count of account blocks have download

	state SyncState

	peers *peerSet

	pending   int // pending count of FileList msg
	responsed int // number of FileList msg received
	mu        sync.Mutex
	fileMap   map[filename]*fileRecord

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
	atomic.AddUint64(&s.aCount, 1)
	return nil
}

func (s *syncer) receiveSnapshotBlock(block *ledger.SnapshotBlock) error {
	err := s.verifier.VerifyNetSb(block)
	if err != nil {
		return err
	}

	s.notifier.notifySnapshotBlock(block, types.RemoteSync)
	atomic.AddUint64(&s.sCount, 1)
	return nil
}

func newSyncer(chain Chain, peers *peerSet, verifier Verifier, gid MsgIder, notifier blockNotifier) *syncer {
	s := &syncer{
		state:     SyncNotStart,
		chain:     chain,
		peers:     peers,
		fileMap:   make(map[filename]*fileRecord),
		eventChan: make(chan peerEvent, 1),
		verifier:  verifier,
		notifier:  notifier,
		subs:      make(map[int]SyncStateCallback),
		log:       log15.New("module", "net/syncer"),
	}

	pool := newChunkPool(peers, gid, s)
	fc := newFileClient(chain, s, peers)
	s.exec = newExecutor(s)

	s.pool = pool
	s.fc = fc

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
	return s.state
}

func (s *syncer) setState(st SyncState) {
	s.state = st
	for _, sub := range s.subs {
		sub(st)
	}
}

func (s *syncer) Stop() {
	if atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		if s.term == nil {
			return
		}

		select {
		case <-s.term:
		default:
			close(s.term)
			s.exec.terminate()
			s.pool.stop()
			s.fc.stop()
			s.clear()
			s.peers.UnSub(s.eventChan)
		}
	}
}

func (s *syncer) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.aCount = 0
	s.sCount = 0
	s.pending = 0
	s.fileMap = make(map[filename]*fileRecord)
}

func (s *syncer) Start() {
	// is running
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}
	s.term = make(chan struct{})

	s.peers.Sub(s.eventChan)

	defer s.Stop()

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
	var lastCheckTime = time.Now()

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
						// no peers, then quit
						return
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

			s.log.Info(fmt.Sprintf("sync current: %d, chain speed %d", current.Height, current.Height-s.current))

			if current.Height == s.current {
				if now.Sub(lastCheckTime) > 10*time.Minute {
					s.setState(Syncerr)
				}
			} else if s.state == Syncing {
				s.current = current.Height
				lastCheckTime = now
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

	s.mu.Lock()
	s.pending = len(l)
	s.mu.Unlock()

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
		p.Report(err)
		return
	} else {
		s.log.Info(fmt.Sprintf("get subledger from %d to %d to %s at %d", from, pTo, p.RemoteAddr(), p.Height()))
	}
}

func (s *syncer) receiveFileList(msg *message.FileList) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, file := range msg.Files {
		if _, ok := s.fileMap[file.Filename]; ok {
			continue
		}

		s.fileMap[file.Filename] = &fileRecord{file, false}
	}

	s.responsed++

	if s.responsed*2 > s.pending || s.responsed > 3 {
		// prepare tasks
		if len(s.fileMap) > 0 {
			s.fc.start()

			// has new file to download
			files := make(Files, len(s.fileMap))
			i := 0
			for _, r := range s.fileMap {
				if !r.add {
					files[i] = r.File
					i++
					r.add = true
				}
			}

			if files = files[:i]; len(files) > 0 {
				sort.Sort(files)
				start := files[0].StartHeight

				// delete following tasks
				start = s.exec.deleteFrom(start)

				// add new tasks
				for _, file := range files {
					if file.EndHeight >= start {
						s.exec.add(&syncTask{
							task: &fileTask{
								file:       file,
								downloader: s.fc,
							},
							typ: syncFileTask,
						})
					}
				}
			}
		} else {
			to, _ := s.exec.end()
			// no tasks, then use chunk
			if to == 0 {
				s.pool.start()

				cks := splitChunk(s.from, s.to, 3600)
				for _, ck := range cks {
					s.exec.add(&syncTask{
						task: &chunkTask{
							from:       ck[0],
							to:         ck[1],
							downloader: s.pool,
						},
						typ: syncChunkTask,
					})
				}
			}
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

		s.receiveFileList(res)

	case SubLedgerCode:
		s.log.Info(fmt.Sprintf("receive %s from %s", SubLedgerCode, sender.RemoteAddr()))
		return s.pool.Handle(msg, sender)
	}

	return nil
}

func (s *syncer) taskDone(t *syncTask, err error) {
	if err != nil {
		s.log.Error(fmt.Sprintf("sync task %s error", t.String()))

		if s.state != Syncing || atomic.LoadInt32(&s.running) == 0 {
			return
		}

		_, to := t.bound()
		target := atomic.LoadUint64(&s.to)

		if to <= target {
			s.setState(Syncerr)
			return
		}
	}
}

func (s *syncer) allTaskDone(last *syncTask) {
	_, to := last.bound()
	target := atomic.LoadUint64(&s.to)

	if to >= target {
		s.setState(SyncDownloaded)
		return
	}

	// use chunk
	cks := splitChunk(to+1, target, 3600)
	for _, ck := range cks {
		s.exec.add(&syncTask{
			task: &chunkTask{
				from:       ck[0],
				to:         ck[1],
				downloader: s.pool,
			},
			typ: syncChunkTask,
		})
	}

	s.pool.start()
}

type SyncStatus struct {
	From     uint64
	To       uint64
	Current  uint64
	Received uint64
	State    SyncState
}

func (s *syncer) Status() SyncStatus {
	current := s.chain.GetLatestSnapshotBlock()

	return SyncStatus{
		From:     s.from,
		To:       s.to,
		Current:  current.Height,
		Received: s.sCount,
		State:    s.state,
	}
}

type SyncDetail struct {
	SyncStatus
	ExecutorStatus
	FileClientStatus
	ChunkPoolStatus
}

func (s *syncer) Detail() SyncDetail {
	return SyncDetail{
		SyncStatus:       s.Status(),
		ExecutorStatus:   s.exec.status(),
		FileClientStatus: s.fc.status(),
		ChunkPoolStatus:  s.pool.status(),
	}
}
