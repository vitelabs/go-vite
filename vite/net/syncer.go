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

	"github.com/vitelabs/go-vite/vite/net/message"

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
const syncTaskSize = 1000
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
	count() int
}

type syncer struct {
	from, to uint64

	state syncState

	timeout time.Duration // sync error if timeout

	peers     syncPeerSet
	eventChan chan peerEvent // get peer add/delete event

	taskCanceled int32
	sk           *skeleton
	syncWG       sync.WaitGroup

	chain      syncChain // query current block and height
	downloader syncDownloader
	reader     syncCacheReader
	irreader   IrreversibleReader

	blackBlocks map[types.Hash]struct{}

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
		s.log.Info(fmt.Sprintf("receive HashHeightList from %s", sender))
		s.sk.receiveHashList(msg, sender)
	}

	return nil
}

func newSyncer(chain syncChain, peers syncPeerSet, reader syncCacheReader, downloader syncDownloader, timeout time.Duration) *syncer {
	s := &syncer{
		chain:       chain,
		peers:       peers,
		eventChan:   make(chan peerEvent, 1),
		subs:        make(map[int]SyncStateCallback),
		downloader:  downloader,
		reader:      reader,
		timeout:     timeout,
		sk:          newSkeleton(peers, new(gid)),
		blackBlocks: make(map[types.Hash]struct{}),
		term:        make(chan struct{}),
		log:         netLog.New("module", "syncer"),
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

	syncPeerHeight := syncPeer.Height()

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
					s.log.Error("sync error: stuck")
					s.stopSync()
					s.syncWG.Wait()

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

func constructTasks(start *ledger.HashHeight, hhs []*message.HashHeightPoint) (ts syncTasks) {
	count := len(hhs) - 1
	ts = make(syncTasks, count)

	for i := 0; i < count; i++ {
		ts[i] = &syncTask{
			Segment: interfaces.Segment{
				Bound:    [2]uint64{start.Height + 1, hhs[i+1].Height},
				PrevHash: start.Hash,
				Hash:     hhs[i+1].Hash,
			},
		}
		start = &(hhs[i+1].HashHeight)
	}

	return
}

// start [start0 = currentHeight / syncTaskSize * syncTaskSize, start1 = start0 - 100, start2 = irrevHeight ]
func (s *syncer) sync() {
	s.syncWG.Add(1)
	defer s.syncWG.Done()

	s.state.sync()
	s.log.Info("syncing")

	var start = make([]*ledger.HashHeight, 0, 3)
	var end uint64
	var err error
	var block *ledger.SnapshotBlock

Start:
	genesis := s.chain.GetGenesisSnapshotBlock()

	block = s.irreader.GetIrreversibleBlock()
	var irrevHeight uint64
	var irrevHash types.Hash
	if block != nil {
		irrevHeight = block.Height / syncTaskSize * syncTaskSize
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
	irrevHeight = block.Height
	irrevHash = block.Hash

	height := s.getHeight()
	startHeight := height / syncTaskSize * syncTaskSize
	addIrreverse := false
	for i := 0; i < 2; i++ {
		if startHeight <= irrevHeight {
			startHeight = irrevHeight

			start = append(start, &ledger.HashHeight{
				Height: irrevHeight,
				Hash:   irrevHash,
			})
			addIrreverse = true
			s.log.Warn(fmt.Sprintf("getIrreversibleBlock: %s/%d", irrevHash, irrevHeight))
			break
		} else {
			block, err = s.chain.GetSnapshotBlockByHeight(startHeight)
			if err != nil || block == nil {
				s.log.Error(fmt.Sprintf("failed to find snapshot block at %d", startHeight))
				// maybe rollback
				time.Sleep(200 * time.Millisecond)
				goto Start
			} else {
				start = append(start, &ledger.HashHeight{
					Height: startHeight,
					Hash:   block.Hash,
				})

				if startHeight > syncTaskSize {
					startHeight -= syncTaskSize
				} else {
					break
				}
			}
		}
	}

	if addIrreverse == false {
		start = append(start, &ledger.HashHeight{
			Height: irrevHeight,
			Hash:   irrevHash,
		})
		s.log.Warn(fmt.Sprintf("getIrreversibleBlock: %s/%d", irrevHash, irrevHeight))
	}

	// bugfix: 691500
	const forkHeight = 691500
	if irrevHeight > forkHeight {
		block, err = s.chain.GetSnapshotBlockByHeight(forkHeight)
		if err != nil {
			panic(fmt.Sprintf("failed to get block at %d", forkHeight))
		} else if block != nil {
			start = append(start, &ledger.HashHeight{
				Height: block.Height,
				Hash:   block.Hash,
			})
		}
	}

	atomic.StoreInt32(&s.taskCanceled, 0)
	first := true

Loop:
	for {
		if atomic.LoadInt32(&s.taskCanceled) == 1 {
			break Loop
		}

		syncPeer := s.peers.syncPeer()
		if syncPeer == nil {
			break Loop
		}

		syncHeight := syncPeer.Height()
		syncHeight = syncHeight / syncTaskSize * syncTaskSize
		atomic.StoreUint64(&s.to, syncHeight)

		end = (startHeight + maxBatchChunkSize) / syncTaskSize * syncTaskSize
		if end > syncHeight {
			end = syncHeight
		}

		if start[0].Height >= end {
			time.Sleep(time.Second)
			continue
		}

		points := s.sk.construct(start, end)
		if len(points) == 0 {
			s.log.Warn(fmt.Sprintf("failed to construct skeleton: %d-%d", start[len(start)-1].Height, end))
			time.Sleep(time.Second)
			continue
		}

		// blackList, exit and sync done
		if s.inBlackList(points) {
			s.state.done()
			s.stop()
			return
		}

		var point *ledger.HashHeight
		for _, point = range start {
			if point.Height+1 == points[0].Height {
				if first {
					s.from = point.Height + 1
				}

				break
			}
		}

		if point != nil {
			s.log.Info(fmt.Sprintf("construct skeleton: %d-%d", point.Height, end))

			first = false
			if tasks := constructTasks(point, points); len(tasks) > 0 {
				tasks = s.reader.compareCache(tasks)
				for _, t := range tasks {
					// could be blocked when downloader tasks queue is full
					if false == s.downloader.download(t, false) && atomic.LoadInt32(&s.taskCanceled) == 1 {
						break Loop
					}
				}
			}

			point = &(points[len(points)-1].HashHeight)
			startHeight = point.Height
			start = start[:1]
			start[0] = &ledger.HashHeight{
				Height: point.Height,
				Hash:   point.Hash,
			}
		} else {
			// wrong points
			time.Sleep(200 * time.Millisecond)
		}
	}

	s.sk.reset()
	s.downloader.cancelAllTasks()
}

func (s *syncer) inBlackList(list []*message.HashHeightPoint) bool {
	for _, h := range list {
		if _, ok := s.blackBlocks[h.Hash]; ok {
			return true
		}
	}
	return false
}

func (s *syncer) stopSync() {
	s.downloader.cancelAllTasks()
	atomic.StoreInt32(&s.taskCanceled, 1)
	s.reader.reset()
}

func (s *syncer) setBlackHashList(list []string) {
	for _, str := range list {
		hash, err := types.HexToHash(str)
		if err != nil {
			panic(fmt.Sprintf("failed to parse BlackBlockHash: %s %v", hash, err))
		}
		s.blackBlocks[hash] = struct{}{}
	}
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
