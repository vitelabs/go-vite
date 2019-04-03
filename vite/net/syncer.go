package net

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/interfaces"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
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
		return "unknown sync state_bak"
	}
	return syncStatus[s]
}

// the minimal height difference between snapshot chain of ours and bestPeer
// if the difference is little than this value, then we deem no need sync
const minHeightDifference = 3600
const waitEnoughPeers = 10 * time.Second
const enoughPeers = 3
const chainGrowInterval = time.Second
const maxDownloadTask = 24 * 7 // one week
const downloadTaskSize = 3600

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

type syncer struct {
	from, to uint64
	current  uint64 // current height

	state SyncState

	peers *peerSet

	// query current block and height
	chain Chain

	// get peer add/delete event
	eventChan chan peerEvent

	// handle blocks
	verifier Verifier
	notifier blockNotifier

	// for sync tasks
	downloader chunkDownloader

	exec syncTaskExecutor

	// for subscribe
	curSubId int
	subs     map[int]SyncStateCallback

	mu sync.Mutex

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

func newSyncer(chain Chain, peers *peerSet, verifier Verifier, notifier blockNotifier) *syncer {
	s := &syncer{
		state:     SyncNotStart,
		chain:     chain,
		peers:     peers,
		eventChan: make(chan peerEvent, 1),
		verifier:  verifier,
		notifier:  notifier,
		subs:      make(map[int]SyncStateCallback),
		log:       log15.New("module", "net/syncer"),
	}

	s.downloader = newFileClient(chain, s, peers)
	s.exec = newExecutor(s)

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
			s.peers.unSub(s.eventChan)
		}
	}
}

func (s *syncer) Start() {
	// is running
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}
	s.term = make(chan struct{})

	s.peers.sub(s.eventChan)

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

PREPARE:
	// for now syncState is SyncNotStart
	syncPeer := s.peers.syncPeer()
	if syncPeer == nil {
		s.setState(Syncerr)
		s.log.Error("sync error: no peers")
		return
	}

	syncPeerHeight := syncPeer.height()

	// compare snapshot chain height
	current := s.chain.GetLatestSnapshotBlock()
	// p is not all enough, no need to sync
	if current.Height+minHeightDifference > syncPeerHeight {
		if current.Height < syncPeerHeight {
			err := syncPeer.send(GetSnapshotBlocksCode, 0, &message.GetSnapshotBlocks{
				From:    ledger.HashHeight{Height: syncPeerHeight},
				Count:   1,
				Forward: true,
			})

			if err != nil {
				syncPeer.catch(err)
				goto PREPARE
			}
		}

		s.log.Info(fmt.Sprintf("sync done: syncPeer %s at %d, our height: %d", syncPeer.Address(), syncPeerHeight, current.Height))
		s.setState(Syncdone)
		return
	}

	s.current = current.Height
	s.from = current.Height + 1
	s.to = syncPeerHeight
	s.current = current.Height
	s.setState(Syncing)
	s.downloadLedger(s.from, s.to)

	// check chain height
	checkChainTicker := time.NewTicker(chainGrowInterval)
	defer checkChainTicker.Stop()
	var lastCheckTime = time.Now()

	for {
		select {
		case <-s.eventChan:
			if syncPeer = s.peers.syncPeer(); syncPeer != nil {
				syncPeerHeight = syncPeer.height()
				if shouldSync(current.Height, syncPeerHeight) {
					s.setTarget(syncPeerHeight)
				} else {
					// no need sync
					s.log.Info(fmt.Sprintf("no need sync to bestPeer %s at %d, our height: %d", syncPeer, syncPeerHeight, current.Height))
					s.setState(Syncdone)
					return
				}
			} else {
				s.log.Error("sync error: no peers")
				s.setState(Syncerr)
				// no peers, then quit
				return
			}

		case now := <-checkChainTicker.C:
			current = s.chain.GetLatestSnapshotBlock()

			if current.Height >= s.to {
				s.log.Info(fmt.Sprintf("sync done, current height: %d", current.Height))
				s.setState(Syncdone)
				return
			}

			s.log.Info(fmt.Sprintf("sync current: %d, chain speed %d", current.Height, current.Height-s.current))

			if current.Height == s.current && now.Sub(lastCheckTime) > 10*time.Minute {
				s.setState(Syncerr)
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

func (s *syncer) readCacheLoop() {
Loop:
	for {
		if s.state == Syncdone {
			return
		}

		cacher := s.chain.GetSyncCache()
		cs := cacher.Chunks()

		if len(cs) == 0 {
			time.Sleep(3 * time.Second)
			continue
		}

		var err error
		var reader interfaces.ReadCloser

		c := cs[0]
		if c[1] > s.current && c[1] < s.current+downloadTaskSize {
			reader, err = cacher.NewReader(c[0], c[1])
			if err != nil {
				s.log.Error(fmt.Sprintf("read chunk %d-%d from cache error: %v", c[0], c[1], err))
				goto Loop
			}

			for {
				var ab *ledger.AccountBlock
				var sb *ledger.SnapshotBlock
				ab, sb, err = reader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					s.log.Error(fmt.Sprintf("read chunk %d-%d from cache error: %v", c[0], c[1], err))
					break
				} else if ab != nil {
					err = s.receiveAccountBlock(ab)
					if err != nil {
						s.log.Error(fmt.Sprintf("handle account block %s from cache error: %v", ab.Hash, err))
					}
					break
				} else if sb != nil {
					err = s.receiveSnapshotBlock(sb)
					if err != nil {
						s.log.Error(fmt.Sprintf("handle snapshot block %s from cache error: %v", sb.Hash, err))
					}
					break
				}
			}

			_ = reader.Close()
		}
	}
}

func (s *syncer) downloadLedger(from, to uint64) {
	to2 := from + maxDownloadTask*downloadTaskSize - 1
	if to2 > to {
		to2 = to
	}

	cs := splitChunk(from, to2, downloadTaskSize)

	for _, c := range cs {
		s.exec.add(&syncTask{
			task: &chunkTask{
				from:       c[0],
				to:         c[1],
				downloader: s.downloader,
			},
			typ: syncChunkTask,
		})
	}
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

	// download next batch
	s.downloadLedger(to+1, s.to)
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
		From:    s.from,
		To:      s.to,
		Current: current.Height,
		State:   s.state,
	}
}

type SyncDetail struct {
	SyncStatus
	ExecutorStatus
}

func (s *syncer) Detail() SyncDetail {
	return SyncDetail{
		SyncStatus:     s.Status(),
		ExecutorStatus: s.exec.status(),
	}
}
