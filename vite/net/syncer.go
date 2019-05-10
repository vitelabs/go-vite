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

	"github.com/vitelabs/go-vite/p2p"

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
const maxBatchChunkSize = 100 * syncTaskSize

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

type syncPeerSet interface {
	sub(ch chan<- peerEvent)
	unSub(ch chan<- peerEvent)
	syncPeer() Peer
	bestPeer() Peer
	pick(height uint64) (l []Peer)
}

type syncer struct {
	from, to uint64

	state syncState

	timeout time.Duration // sync error if timeout

	peers     syncPeerSet
	eventChan chan peerEvent // get peer add/delete event

	cancel      chan struct{}
	batchHead   types.Hash
	batchHeight uint64
	sk          *skeleton

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

func (s *syncer) codes() []p2p.Code {
	return []p2p.Code{p2p.CodeHashList}
}

func (s *syncer) handle(msg p2p.Msg, sender Peer) (err error) {
	switch msg.Code {
	case p2p.CodeHashList:
		s.sk.receiveHashList(msg, sender)
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
		sk:         newSkeleton(chain, peers, new(gid)),
		log:        netLog.New("module", "syncer"),
		cancel:     make(chan struct{}),
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

	syncPeerHeight := syncPeer.Height()

	// compare snapshot chain height and local cache height
	var current uint64
	// syncPeer is not all enough, no need to sync
	if current = s.getHeight(); !shouldSync(current, syncPeerHeight) {
		s.log.Info(fmt.Sprintf("sync done: syncPeer %s at %d, our height: %d", syncPeer.String(), syncPeerHeight, current))
		s.state.done()
		return
	}

	s.state.sync()
	s.from = current + 1
	s.to = syncPeerHeight
	go s.sync()

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
				if current = s.getHeight(); current >= bestPeer.Height() {
					// we are taller than the best peer, no need sync
					s.log.Info(fmt.Sprintf("sync done: bestPeer %s at %d, our height: %d", bestPeer.String(), bestPeer.Height(), current))
					s.state.done()

					// cancel rest tasks
					s.cancelDownload()
					s.reader.clean()
					return
				}
			} else {
				s.log.Error("sync error: no peers")
				s.state.error(syncErrorNoPeers)
				// no peers
				// cancel rest tasks, keep cache
				s.cancelDownload()
				s.reader.reset()
				return
			}

		case now := <-checkChainTicker.C:
			current = s.getHeight()
			if current >= s.to {
				s.state.done()

				// clean cache, cancel rest tasks
				s.cancelDownload()
				s.reader.clean()
				return
			}

			if current == oldHeight {
				if now.Sub(lastCheckTime) > s.timeout {
					s.log.Error("sync error: stuck")
					s.state.error(syncErrorStuck)
					// watching chain grow through fetcher
					// clean cache, cancel rest tasks
					s.cancelDownload()
					s.reader.clean()
				}
			} else {
				oldHeight = current
				lastCheckTime = now
			}

		case <-s.term:
			s.state.cancel()

			// cancel rest tasks, keep cache
			s.cancelDownload()
			s.reader.reset()
			return
		}
	}
}

func (s *syncer) getHeight() uint64 {
	return s.chain.GetLatestSnapshotBlock().Height
}

func (s *syncer) chooseBatch(syncHeight uint64) (start []*ledger.HashHeight, end uint64) {
	start = append(start, &ledger.HashHeight{
		Height: s.batchHeight,
		Hash:   s.batchHead,
	})

	end = s.batchHeight + maxBatchChunkSize - 1
	if end > syncHeight {
		end = syncHeight
	}

	return
}

func constructTasks(start *ledger.HashHeight, hhs []*ledger.HashHeight) (ts syncTasks) {
	count := len(hhs) - 1
	ts = make(syncTasks, count)

	for i := 0; i < count; i++ {
		ts[i] = &syncTask{
			from:     start.Height + 1,
			to:       hhs[i+1].Height,
			prevHash: start.Hash,
			endHash:  hhs[i+1].Hash,
		}
		start = hhs[i+1]
	}

	return
}

func (s *syncer) sync() {
Start:
	genesis := s.chain.GetGenesisSnapshotBlock()

	s.batchHeight = s.getHeight() / syncTaskSize * syncTaskSize
	// local chain is too low
	if s.batchHeight <= genesis.Height {
		s.batchHeight = genesis.Height
		s.batchHead = genesis.Hash
	} else {
		block, err := s.chain.GetSnapshotBlockByHeight(s.batchHeight)
		if err != nil || block == nil {
			// maybe rollback
			time.Sleep(200 * time.Millisecond)
			goto Start
		} else {
			s.batchHead = block.Hash
		}
	}

	s.from = s.batchHeight + 1
	cacheCompared := false

	for {
		select {
		case <-s.cancel:
			return
		default:

		}

		syncPeer := s.peers.syncPeer()
		if syncPeer == nil {
			return
		}

		syncHeight := syncPeer.Height()
		syncHeight = syncHeight / syncTaskSize * syncTaskSize
		atomic.StoreUint64(&s.to, syncHeight)

		start, end := s.chooseBatch(syncHeight)
		points := s.sk.construct(start, end)
		if len(points) == 0 {
			time.Sleep(time.Second)
			continue
		}

		var point *ledger.HashHeight
		for _, point = range start {
			if point.Height+1 == points[0].Height {
				break
			}
		}

		if point != nil {
			tasks := constructTasks(point, points)
			if false == cacheCompared {
				tasks = s.reader.compareCache(tasks)
				cacheCompared = true
			}

			s.runTasks(tasks)

			point = points[len(points)-1]
			s.batchHeight = point.Height
			s.batchHead = point.Hash
		} else {
			// wrong points
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (s *syncer) runTasks(ts syncTasks) {
	for _, t := range ts {
		if false == s.downloader.download(t, false) {
			s.log.Warn(fmt.Sprintf("failed to download %s", t))
			break
		}
	}
}

func (s *syncer) cancelDownload() {
	local := s.getHeight()

	s.downloader.cancel(local)

	s.cancel <- struct{}{}
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
	Cache  interfaces.SegmentList  `json:"cache"`
	Chunks [][2]*ledger.HashHeight `json:"chunks"`
}

func (s *syncer) Detail() SyncDetail {
	return SyncDetail{
		SyncStatus:       s.Status(),
		DownloaderStatus: s.downloader.status(),
		Cache:            s.reader.chunks(),
		Chunks:           s.reader.queue(),
	}
}
