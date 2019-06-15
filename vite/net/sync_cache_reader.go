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
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

var errReaderStopped = errors.New("cache reader stopped")
var errReaderInterrupt = errors.New("cache reader interrupt")

const maxQueueSize = 10 << 20 // 10MB
const maxQueueLength = 5

type syncCacheReader interface {
	ChunkReader
	start()
	stop()
	compareCache(ts syncTasks) syncTasks
	chunks() [][2]*ledger.HashHeight
	cache(from, to uint64) interfaces.SegmentList
	caches() interfaces.SegmentList
	reset()
}

type Chunks []*Chunk

func (cs Chunks) Len() int {
	return len(cs)
}

func (cs Chunks) Less(i, j int) bool {
	return cs[i].SnapshotRange[0].Height < cs[j].SnapshotRange[0].Height
}

func (cs Chunks) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

type cacheReader struct {
	chain      syncChain
	verifier   Verifier
	downloader syncDownloader
	irreader   IrreversibleReader

	running bool
	mu      sync.Mutex
	cond    *sync.Cond

	readHeight uint64

	readable int32

	buffer Chunks

	downloadRecord map[interfaces.Segment]peerId

	blackBlocks map[types.Hash]struct{}

	wg  sync.WaitGroup
	log log15.Logger
}

func (s *cacheReader) cache(from, to uint64) (ret interfaces.SegmentList) {
	cs := s.localChunk()
	return retrieveSegments(cs, from, to)
}

func retrieveSegments(cs interfaces.SegmentList, from, to uint64) (ret interfaces.SegmentList) {
	if len(cs) == 0 {
		return nil
	}

	index := sort.Search(len(cs), func(i int) bool {
		return cs[i].Bound[1] >= from
	})

	for i := index; i < len(cs); i++ {
		if cs[i].Bound[0] > to {
			break
		}
		ret = append(ret, cs[i])
	}

	return
}

func (s *cacheReader) caches() interfaces.SegmentList {
	cs := s.localChunk()
	return mergeSegments(cs)
}

func mergeSegments(cs interfaces.SegmentList) (cs2 interfaces.SegmentList) {
	if len(cs) == 0 {
		return
	}

	segment := cs[0]
	for i := 1; i < len(cs); i++ {
		if cs[i].PrevHash == segment.Hash && cs[i].Bound[0] == segment.Bound[1]+1 {
			segment.Hash = cs[i].Hash
			segment.Bound[1] = cs[i].Bound[1]
		} else {
			cs2 = append(cs2, segment)
			segment = cs[i]
		}
	}

	return
}

func (s *cacheReader) chunks() (ret [][2]*ledger.HashHeight) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret = make([][2]*ledger.HashHeight, len(s.buffer))
	for i, c := range s.buffer {
		ret[i] = c.SnapshotRange
	}

	return
}

func (s *cacheReader) Peek() (c *Chunk) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.buffer) > 0 {
		c = s.buffer[0]
		s.log.Info(fmt.Sprintf("peek cache %d-%d", c.SnapshotRange[0].Height, c.SnapshotRange[1].Height))
	}

	return
}

func (s *cacheReader) Pop(endHash types.Hash) {
	s.mu.Lock()
	if len(s.buffer) > 0 {
		if s.buffer[0].SnapshotRange[1].Hash == endHash {
			s.log.Info(fmt.Sprintf("pop cache %d-%d %s", s.buffer[0].SnapshotRange[0].Height, s.buffer[0].SnapshotRange[1].Height, endHash))

			s.buffer = s.buffer[1:]

			s.cond.Signal()
		}
	}
	s.mu.Unlock()
}

//func (s *cacheReader) bufferSize() (n int64) {
//	for _, c := range s.buffer {
//		n += c.size
//	}
//
//	return
//}

// will blocked if queue is full
func (s *cacheReader) addChunkToBuffer(c *Chunk) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if false == s.running || s.canRead() == false {
			return
		}

		if len(s.buffer) > maxQueueLength {
			s.cond.Wait()
		} else {
			break
		}
	}

	s.buffer = append(s.buffer, c)
}

func newCacheReader(chain syncChain, verifier Verifier, downloader syncDownloader) *cacheReader {
	s := &cacheReader{
		chain:          chain,
		verifier:       verifier,
		downloader:     downloader,
		buffer:         make(Chunks, 0, maxQueueLength),
		log:            netLog.New("module", "cache"),
		readable:       1,
		downloadRecord: make(map[interfaces.Segment]peerId),
		blackBlocks:    make(map[types.Hash]struct{}),
	}

	s.cond = sync.NewCond(&s.mu)

	downloader.addListener(s.chunkDownloaded)

	return s
}

func (s *cacheReader) getHeight() uint64 {
	return s.chain.GetLatestSnapshotBlock().Height
}

func (s *cacheReader) start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.readLoop()

	s.wg.Add(1)
	go s.cleanLoop()
}

func (s *cacheReader) stop() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	s.reset()

	s.wg.Wait()
}

// from low to high
func compareCache(cs interfaces.SegmentList, ts syncTasks, catch func(segment interfaces.Segment, t *syncTask)) syncTasks {
	var i int
	var t *syncTask
	var shouldRemoved bool
Loop:
	for _, c := range cs {
		for i < len(ts) {
			t = ts[i]
			if c.Bound[1] < t.Bound[0] {
				continue Loop
			}

			if c.Bound[0] > t.Bound[1] {
				i++
				continue
			}

			// task and segment is overlapped
			if c != t.Segment {
				catch(c, t)
				continue Loop
			}

			// task have the same cache
			ts[i] = nil
			shouldRemoved = true
			i++
		}
	}

	if shouldRemoved {
		i = 0
		for j := 0; j < len(ts); j++ {
			if ts[j] != nil {
				ts[i] = ts[j]
				i++
			}
		}

		ts = ts[:i]
	}

	return ts
}

func (s *cacheReader) compareCache(ts syncTasks) syncTasks {
	s.pause()
	defer s.resume() // will signal reader

	if s.readHeight == 0 {
		s.readHeight = ts[0].Bound[0] - 1
	}

	cs := s.localChunk()
	if len(cs) == 0 {
		return ts
	}

	last := cs[len(cs)-1]
	if last.Bound[1] < ts[0].Bound[0] {
		return ts
	}

	// chunk and tasks are overlapped
	ts = compareCache(cs, ts, s.deleteChunk)

	return ts
}

func (s *cacheReader) deleteChunk(segment interfaces.Segment, t *syncTask) {
	cache := s.chain.GetSyncCache()
	_ = cache.Delete(segment)

	s.log.Warn(fmt.Sprintf("delete chunk %d-%d/%s/%s, task %d-%d/%s/%s", segment.Bound[0], segment.Bound[1], segment.PrevHash, segment.Hash, t.Bound[0], t.Bound[1], t.PrevHash, t.Hash))

	// chunk has been read to buffer
	if segment.Bound[1] <= s.readHeight {
		s.mu.Lock()
		for i, c := range s.buffer {
			if c.SnapshotRange[1].Height == segment.Bound[1] {
				s.buffer = s.buffer[:i]

				s.readHeight = c.SnapshotRange[0].Height
				if s.readHeight >= t.Bound[0] {
					s.readHeight = t.Bound[0] - 1
				}

				s.log.Warn(fmt.Sprintf("delete chunk in buffer from %d", i))

				// signal reader, maybe blocked when addChunkToBuffer
				s.cond.Signal()

				break
			}
		}
		s.mu.Unlock()
	}
}

func (s *cacheReader) localChunk() interfaces.SegmentList {
	return s.chain.GetSyncCache().Chunks()
}

func (s *cacheReader) chunkDownloaded(t syncTask, err error) {
	if err == nil {
		s.mu.Lock()
		s.downloadRecord[t.Segment] = t.source
		s.mu.Unlock()
	}
}

func (s *cacheReader) chunkReadFailed(segment interfaces.Segment, fatal bool) {
	if fatal {
		s.mu.Lock()
		id := s.downloadRecord[segment]
		s.mu.Unlock()

		s.downloader.addBlackList(id)
		s.log.Warn(fmt.Sprintf("block sync peer: %s", id))

		cache := s.chain.GetSyncCache()
		err := cache.Delete(segment)
		if err == nil {
			s.downloader.download(&syncTask{
				Segment: segment,
			}, true)
		}
	}
}

func (s *cacheReader) reset() {
	s.pause()

	s.mu.Lock()
	s.readHeight = 0
	s.buffer = s.buffer[:0]
	s.mu.Unlock()

	s.cond.Signal()
}

func (s *cacheReader) removeUselessChunks(cleanWrong bool) {
	height := s.getHeight()
	cache := s.chain.GetSyncCache()
	cs := cache.Chunks()
	genesisHeight := s.chain.GetGenesisSnapshotBlock().Height

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range cs {
		if c.Bound[1] > height {
			if cleanWrong {
				if c.Bound[0]%syncTaskSize != 1 && c.Bound[0] != genesisHeight+1 {
					_ = cache.Delete(c)
				} else if c.Bound[1]%syncTaskSize != 0 {
					_ = cache.Delete(c)
				}
			} else {
				break
			}
		} else {
			delete(s.downloadRecord, c)
			_ = cache.Delete(c)
		}
	}
}

func (s *cacheReader) read(c interfaces.Segment) (chunk *Chunk, fatal bool, err error) {
	fatal = true

	cache := s.chain.GetSyncCache()
	reader, err := cache.NewReader(c)
	if err != nil {
		return
	}

	verified := reader.Verified()

	chunk = newChunk(c.PrevHash, c.Bound[0]-1, c.Hash, c.Bound[1], types.RemoteSync)
	chunk.size = reader.Size()

	var ab *ledger.AccountBlock
	var sb *ledger.SnapshotBlock
	for {
		if false == s.running || false == s.canRead() {
			_ = reader.Close()
			fatal = false
			if false == s.running {
				err = errReaderStopped
			} else {
				err = errReaderInterrupt
			}
			return
		}

		ab, sb, err = reader.Read()
		if err != nil {
			break
		} else if ab != nil {
			if _, ok := s.blackBlocks[ab.Hash]; ok {
				s.log.Warn(fmt.Sprintf("accountblock %s is in blacklist", ab.Hash))
				break
			}

			if verified == false {
				if err = s.verifier.VerifyNetAb(ab); err != nil {
					break
				}
			}

			if err = chunk.addAccountBlock(ab); err != nil {
				break
			}

		} else if sb != nil {
			if _, ok := s.blackBlocks[sb.Hash]; ok {
				s.log.Warn(fmt.Sprintf("snapshotblock %s is in blacklist", sb.Hash))
				break
			}

			if verified == false {
				if err = s.verifier.VerifyNetSb(sb); err != nil {
					break
				}
			}

			if err = chunk.addSnapshotBlock(sb); err != nil {
				break
			}
		}
	}

	_ = reader.Close()

	if err == io.EOF {
		err = chunk.done()
	}

	// no error, set reader verified
	if err == nil {
		reader.Verify()
	}

	return
}

func (s *cacheReader) pause() {
	atomic.StoreInt32(&s.readable, 0)
}

func (s *cacheReader) resume() {
	atomic.StoreInt32(&s.readable, 1)
	s.cond.Signal()
}

func (s *cacheReader) canRead() bool {
	return atomic.LoadInt32(&s.readable) == 1
}

func (s *cacheReader) readLoop() {
	defer s.wg.Done()

	s.readHeight = 0

	cache := s.chain.GetSyncCache()

	var cs interfaces.SegmentList

Loop:
	for {
		if false == s.running {
			break Loop
		}

		cs = cache.Chunks()
		if len(cs) == 0 || s.canRead() == false {
			time.Sleep(time.Second)
			continue
		}

		// read chunks
		for _, c := range cs {
			// chunk has read
			if c.Bound[1] <= s.readHeight {
				continue
			}

			// missing chunk
			if c.Bound[0] > s.readHeight+1 {
				time.Sleep(200 * time.Millisecond)
				// chunk downloaded
				continue Loop
			}

			s.log.Info(fmt.Sprintf("begin read cache %d-%d", c.Bound[0], c.Bound[1]))
			chunk, fatal, err := s.read(c)

			if err != nil {
				// read chunk error
				s.log.Error(fmt.Sprintf("failed to read cache %d-%d: %v", c.Bound[0], c.Bound[1], err))
				s.chunkReadFailed(c, fatal)
			} else {
				s.log.Info(fmt.Sprintf("read cache %d-%d done", c.Bound[0], c.Bound[1]))
				// set readHeight should be very seriously
				s.readHeight = c.Bound[1]
				// will be block
				s.addChunkToBuffer(chunk)
			}
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func (s *cacheReader) cleanLoop() {
	defer s.wg.Done()

	cache := s.chain.GetSyncCache()

	var initDuration = 10 * time.Second
	var maxDuration = 10 * time.Minute
	var duration = initDuration

	var cs interfaces.SegmentList
	var irevBlock *ledger.SnapshotBlock

Loop:
	for {
		if false == s.running {
			break Loop
		}

		irevBlock = s.irreader.GetIrreversibleBlock()
		cs = cache.Chunks()
		if len(cs) == 0 || irevBlock == nil {
			duration *= 2
			if duration > maxDuration {
				duration = initDuration
			}
			time.Sleep(duration)
			continue
		}

		// read chunks
		for _, c := range cs {
			if c.Bound[1] < irevBlock.Height {
				_ = cache.Delete(c)
			}
		}

		time.Sleep(duration)
	}
}

func (s *cacheReader) setBlackHashList(list []string) {
	for _, str := range list {
		hash, err := types.HexToHash(str)
		if err != nil {
			panic(fmt.Sprintf("failed to parse BlackBlockHash: %s %v", hash, err))
		}
		s.blackBlocks[hash] = struct{}{}
	}
}
