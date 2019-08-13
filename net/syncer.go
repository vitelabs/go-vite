/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package net

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

// the minimal height difference between snapshot chain of ours and bestPeer,
// if the difference is little than this value, then we deem no need sync.
const minHeightDifference = 100
const waitEnoughPeers = 5 * time.Second
const enoughPeers = 3
const checkChainInterval = time.Second
const syncTaskSize = 100
const maxBatchChunkSize = 500 * syncTaskSize

func shouldSync(from, to uint64) bool {
	if to >= from+minHeightDifference {
		return true
	}

	return false
}

type syncer struct {
	sbp bool

	syncing int32

	from, to uint64

	state syncState

	timeout time.Duration // sync error if timeout

	peers     *peerSet
	eventChan chan peerEvent // get peer add/delete event

	taskCanceled int32
	sk           *skeleton
	syncWG       sync.WaitGroup

	chain      syncChain // query current block and height
	downloader syncDownloader
	reader     syncCacheReader
	irreader   IrreversibleReader

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

func (s *syncer) codes() []Code {
	return []Code{CodeHashList}
}

func (s *syncer) handle(msg Msg) (err error) {
	switch msg.Code {
	case CodeHashList:
		s.log.Info(fmt.Sprintf("receive HashHeightList from %s", msg.Sender))
		s.sk.receiveHashList(msg, msg.Sender)
	}

	return nil
}

func newSyncer(chain syncChain, peers *peerSet, reader syncCacheReader, downloader syncDownloader, irreader IrreversibleReader, timeout time.Duration, blackBlocks map[types.Hash]struct{}) *syncer {
	if len(blackBlocks) == 0 {
		blackBlocks = make(map[types.Hash]struct{})
	}

	s := &syncer{
		sbp:          false,
		syncing:      0,
		from:         0,
		to:           0,
		state:        nil,
		timeout:      timeout,
		peers:        peers,
		eventChan:    make(chan peerEvent, 1),
		taskCanceled: 0,
		sk:           newSkeleton(peers, new(gid), blackBlocks),
		syncWG:       sync.WaitGroup{},
		chain:        chain,
		downloader:   downloader,
		reader:       reader,
		irreader:     irreader,
		curSubId:     0,
		subs:         make(map[int]SyncStateCallback),
		mu:           sync.Mutex{},
		running:      0,
		term:         make(chan struct{}),
		log:          netLog.New("module", "syncer"),
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

func (s *syncer) checkLoop(run *int32) {
	checkTicker := time.NewTicker(3 * time.Second)
	defer checkTicker.Stop()

	for {
		if *run == 0 {
			return
		}

		<-checkTicker.C

		current := s.chain.GetLatestSnapshotBlock().Height
		syncPeer := s.peers.syncPeer()
		if syncPeer == nil {
			continue
		}

		if atomic.LoadInt32(&s.running) == 1 {
			continue
		}

		if shouldSync(current, syncPeer.Height) || s.state.state() != SyncDone {
			go s.start()
		}
	}
}

func (s *syncer) stop() {
	if atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		s.peers.unSub(s.eventChan)
		term := s.term
		select {
		case <-term:
		default:
			close(term)
		}
		s.stopSync()
	}
}

func (s *syncer) start() {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}

	s.term = make(chan struct{})

	s.peers.sub(s.eventChan)

	defer s.stop()

	if s.peers.count() < enoughPeers {
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
	}

	var retrySync int

Prepare:
	syncPeer := s.peers.syncPeer()
	// all peers disconnected
	if syncPeer == nil {
		s.log.Error("sync error: no peers")
		s.state.error(syncErrorNoPeers)
		return
	}

	syncPeerHeight := syncPeer.Height

	// compare snapshot chain height and local cache height
	var current uint64
	// syncPeer is not all enough, no need to sync
	if current = s.getHeight(); !shouldSync(current, syncPeerHeight) {
		s.log.Info(fmt.Sprintf("sync done: syncPeer %s at %d, our height: %d", syncPeer.String(), syncPeerHeight, current))
		s.state.done()
		return
	}

	// prevent chain check
	s.from = current + 1
	s.to = syncPeerHeight

	if err := s.sync(); err == nil {
		s.state.sync()
		s.log.Info(fmt.Sprintf("syncing: local %d to %d", current, syncPeerHeight))
	} else {
		s.log.Warn(fmt.Sprintf("failed to sync local %d to %d: %v", current, syncPeerHeight, err))
		return
	}

	// check chain height
	checkChainTicker := time.NewTicker(checkChainInterval)
	defer checkChainTicker.Stop()
	var lastCheckTime = time.Now()
	var oldHeight = current

	for {
		select {
		case <-s.eventChan:
			// peers change, find new peer or peer disconnected. Choose sync peer again.
			if bestPeer := s.peers.bestPeer(); bestPeer != nil {
				if current = s.getHeight(); current >= bestPeer.Height {
					// we are taller than the best peer, no need sync
					s.log.Info(fmt.Sprintf("sync done: bestPeer %s at %d, our height: %d", bestPeer.String(), bestPeer.Height, current))
					s.state.done()
					return
				}
			} else {
				s.log.Error("sync error: no peers")
				s.state.error(syncErrorNoPeers)
				return
			}

		case now := <-checkChainTicker.C:
			current = s.getHeight()
			if current >= s.to {
				s.state.done()
				return
			}

			if current == oldHeight {
				if now.Sub(lastCheckTime) > s.timeout {
					s.log.Error(fmt.Sprintf("sync error: stuck at %d", current))
					s.stopSync()

					if retrySync > 3 {
						s.state.error(syncErrorStuck)
					} else {
						retrySync++
						goto Prepare
					}
				}
			} else {
				oldHeight = current
				lastCheckTime = now
			}

		case <-s.term:
			s.state.cancel()
			return
		}
	}
}

func (s *syncer) getHeight() uint64 {
	return s.chain.GetLatestSnapshotBlock().Height
}

// eg. current height is 768, IrreversibleBlock is 666, then return [700, 600]
func (s *syncer) getInitStart() (start []*ledger.HashHeight) {
	start = make([]*ledger.HashHeight, 0, 3)
	var err error
	var block *ledger.SnapshotBlock

	genesis := s.chain.GetGenesisSnapshotBlock()

Start:
	var irrevPoint *ledger.HashHeight
	// compare IrreversibleBlock and genesis block
	// set the real IrreversibleBlock
	block = s.irreader.GetIrreversibleBlock()
	if block != nil {
		irrevHeight := block.Height / syncTaskSize * syncTaskSize
		if irrevHeight > genesis.Height {
			block, err = s.chain.GetSnapshotBlockByHeight(irrevHeight)
			if err != nil || block == nil {
				s.log.Error(fmt.Sprintf("failed to find snapshot block at %d", irrevHeight))
				block = genesis
			}
		} else {
			block = genesis
		}
	} else {
		block = genesis
	}
	irrevPoint = &ledger.HashHeight{block.Height, block.Hash}

	height := s.getHeight() / syncTaskSize * syncTaskSize
	for i := 0; i < 2; i++ {
		if height <= irrevPoint.Height {
			start = append(start, irrevPoint)
			return
		} else {
			block, err = s.chain.GetSnapshotBlockByHeight(height)
			if err != nil || block == nil {
				s.log.Error(fmt.Sprintf("failed to find snapshot block at %d", height))
				// maybe rollback
				goto Start
			} else {
				start = append(start, &ledger.HashHeight{
					Height: height,
					Hash:   block.Hash,
				})

				if height > syncTaskSize {
					height -= syncTaskSize
				}
			}
		}
	}

	start = append(start, irrevPoint)

	return
}

func (s *syncer) getEnd(start []*ledger.HashHeight) (end uint64) {
	syncPeer := s.peers.syncPeer()
	if syncPeer == nil {
		return 0
	}

	syncHeight := syncPeer.Height
	atomic.StoreUint64(&s.to, syncHeight)

	startHeight := start[0].Height
	end = (startHeight + maxBatchChunkSize) / maxBatchChunkSize * maxBatchChunkSize
	if end > syncHeight {
		end = syncHeight
	}

	return
}

func (s *syncer) getHashHeightList(start []*ledger.HashHeight, end uint64) (list []*HashHeightPoint, err error) {
	if start[0].Height >= end {
		return nil, fmt.Errorf("start %d is not small than end %d", start[0].Height, end)
	}

	s.sk.reset()

	return s.sk.construct(start, end), nil
}

func (s *syncer) verifyHashHeightList(start []*ledger.HashHeight, points []*HashHeightPoint) (startPoint *ledger.HashHeight, err error) {
	if len(points) == 0 {
		return nil, errors.New("hash height list is nil")
	}

	var point *ledger.HashHeight
	for _, point = range start {
		if point.Height+1 == points[0].Height {
			startPoint = point
			break
		}
	}

	if startPoint == nil {
		return nil, errors.New("skeleton has no right start point")
	}

	return
}

// start [start0 = currentHeight / syncTaskSize * syncTaskSize, start1 = start0 - 100, start2 = irrevHeight ]
func (s *syncer) sync() error {
	// get hash height list first
	start := s.getInitStart()
	end := s.getEnd(start)

	// construct hash height list
	points, err := s.getHashHeightList(start, end)
	if err != nil {
		return fmt.Errorf("failed to get hashheight list: %v", err)
	}

	startPoint, err := s.verifyHashHeightList(start, points)
	if err != nil {
		return err
	}

	s.reader.reset()
	s.from = startPoint.Height + 1
	go s.downloadLoop(startPoint, end, points)

	return nil
}

// init points will be construct before download
func (s *syncer) downloadLoop(point *ledger.HashHeight, end uint64, points []*HashHeightPoint) {
	s.syncWG.Add(1)
	defer s.syncWG.Done()

	atomic.StoreInt32(&s.taskCanceled, 0)

	var err error

Loop:
	for {
		s.log.Info(fmt.Sprintf("construct skeleton: %d-%d", point.Height, end))

		if tasks := s.reader.compareCache(point, points); len(tasks) > 0 {
			for _, t := range tasks {
				// could be blocked when downloader tasks queue is full
				if false == s.downloader.download(t, false) || atomic.LoadInt32(&s.taskCanceled) == 1 {
					s.log.Warn("break download loop")
					s.downloader.cancelAllTasks()
					break Loop
				}
			}
		}

		// the last task has been submitted
		if end%syncTaskSize != 0 {
			s.log.Info(fmt.Sprintf("download loop done at end: %d", end))
			return
		}

		// continue download
		point = &(points[len(points)-1].HashHeight)
		start := []*ledger.HashHeight{
			{point.Height, point.Hash},
		}
		end = s.getEnd(start)

		points, err = s.getHashHeightList(start, end)
		if err != nil {
			s.log.Warn(fmt.Sprintf("failed to get hash height list: %v", err))
			return
		}

		if _, err = s.verifyHashHeightList(start, points); err != nil {
			s.log.Warn(fmt.Sprintf("failed to verify hash height list: %v", err))
			return
		}
	}
}

func (s *syncer) stopSync() {
	s.log.Warn("stop sync")
	atomic.StoreInt32(&s.taskCanceled, 1)
	s.downloader.cancelAllTasks() // cancel first, because downloadLoop is blocked when download task queue is full
	s.reader.reset()
	s.syncWG.Wait()
}

type SyncStatus struct {
	From    uint64    `json:"from"`
	To      uint64    `json:"to"`
	Current uint64    `json:"current"`
	State   SyncState `json:"state"`
	Status  string    `json:"status"`
}

func (s *syncer) Status() SyncStatus {
	st := s.state.state()
	return SyncStatus{
		From:    s.from,
		To:      s.to,
		Current: s.getHeight(),
		State:   st,
		Status:  st.String(),
	}
}

type SyncDetail struct {
	SyncStatus
	DownloaderStatus
	Chunks [][2]*ledger.HashHeight `json:"chunks"`
	Caches interfaces.SegmentList  `json:"caches"`
}

func (s *syncer) Detail() SyncDetail {
	return SyncDetail{
		SyncStatus:       s.Status(),
		DownloaderStatus: s.downloader.status(),
		Chunks:           s.reader.chunks(),
		Caches:           s.reader.caches(),
	}
}
