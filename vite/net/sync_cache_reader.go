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
	"sync"
	"time"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

var errNoSnapshotBlocksInChunk = errors.New("no snapshot blocks")
var errReaderStopped = errors.New("cache reader stopped")

const maxQueueSize = 10 << 20 // 10MB
const maxQueueLength = 5

type syncCacheReader interface {
	ChunkReader
	start()
	stop()
	compareCache(ts syncTasks) syncTasks
	clean() // clean cache and reset
	reset() // reset state (readTo and requestTo)
	chunks() interfaces.SegmentList
	queue() [][2]*ledger.HashHeight
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

	running bool
	mu      sync.Mutex
	cond    *sync.Cond

	readHeight   uint64
	readHash     types.Hash
	initReadDone chan struct{}

	buffer Chunks
	index  int

	wg  sync.WaitGroup
	log log15.Logger
}

func (s *cacheReader) queue() (ret [][2]*ledger.HashHeight) {
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

	if s.index < len(s.buffer) {
		c = s.buffer[s.index]
		s.index++
	}

	return
}

func (s *cacheReader) Pop(endHash types.Hash) {
	s.log.Info(fmt.Sprintf("pop cache %s", endHash))

	s.mu.Lock()
	if len(s.buffer) > 0 {
		if s.buffer[0].SnapshotRange[1].Hash == endHash {
			s.buffer = s.buffer[1:]
			s.index--
		}
	}
	s.mu.Unlock()

	s.cond.Signal()
}

func (s *cacheReader) bufferSize() (n int64) {
	for _, c := range s.buffer {
		n += c.size
	}

	return
}

// will blocked if queue is full
func (s *cacheReader) addChunkToBuffer(c *Chunk) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if false == s.running {
			return
		}

		if s.bufferSize() >= maxQueueSize || len(s.buffer) > maxQueueLength {
			s.cond.Wait()
		} else {
			break
		}
	}

	s.buffer = append(s.buffer, c)
}

func newCacheReader(chain syncChain, verifier Verifier, downloader syncDownloader) syncCacheReader {
	s := &cacheReader{
		chain:        chain,
		verifier:     verifier,
		downloader:   downloader,
		buffer:       make(Chunks, 0, maxQueueLength),
		log:          netLog.New("module", "cache"),
		initReadDone: make(chan struct{}, 1),
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

	go s.initReadChunks()
}

func (s *cacheReader) initReadChunks() {
	current := s.chain.GetLatestSnapshotBlock()
	s.readHeight = current.Height
	s.readHash = current.Hash

	cache := s.chain.GetSyncCache()
	cs := cache.Chunks()

	// set the init state
	for _, c := range cs {
		if c.Bound[1] <= current.Height {
			_ = cache.Delete(c)
			continue
		}

		if c.Bound[0] > current.Height+1 {
			break
		}

		s.readHeight = c.Bound[0] - 1
		s.readHash = c.PrevHash
	}

	// read chunks
	cs = cache.Chunks()
	for _, c := range cs {
		if s.running == false {
			break
		}

		if c.Bound[0] == s.readHeight+1 && c.PrevHash == s.readHash {
			chunk, err := s.read(c)
			if err != nil {
				_ = cache.Delete(c)
				break
			} else {
				s.addChunkToBuffer(chunk)
				s.readHeight = c.Bound[1]
				s.readHash = c.Hash
			}
		} else {
			break
		}
	}

	s.initReadDone <- struct{}{}
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
Loop:
	for _, c := range cs {
		for i < len(ts) {
			t = ts[i]
			if c.Bound[1] < t.from {
				continue Loop
			}

			if c.Bound[0] > t.to {
				i++
				continue
			}

			if c.PrevHash != t.prevHash || c.Bound[0] != t.from || c.Bound[1] != t.to || c.Hash != t.endHash {
				catch(c, t)
				continue Loop
			}

			// task have the same cache
			ts[i] = nil
			i++
		}
	}

	i = 0
	for j := 0; j < len(ts); j++ {
		if ts[j] != nil {
			ts[i] = ts[j]
			i++
		}
	}

	return ts[:i]
}

func (s *cacheReader) compareCache(ts syncTasks) syncTasks {
	cs := s.chunks()

	ts = compareCache(cs, ts, s.deleteInitChunk)

	s.wg.Add(1)
	go s.readLoop()

	s.cond.Signal()

	return ts
}

func (s *cacheReader) deleteInitChunk(segment interfaces.Segment, t *syncTask) {
	cache := s.chain.GetSyncCache()
	_ = cache.Delete(segment)

	if segment.Bound[1] <= s.readHeight {
		s.mu.Lock()
		for i, c := range s.buffer {
			if c.SnapshotRange[1].Height == segment.Bound[1] {
				s.buffer = s.buffer[:i]
				s.readHeight = t.from - 1
				s.readHash = t.prevHash
				s.index = i

				// signal reader
				s.cond.Signal()
				break
			}
		}
		s.mu.Unlock()
	}
}

func (s *cacheReader) chunks() interfaces.SegmentList {
	return s.chain.GetSyncCache().Chunks()
}

func (s *cacheReader) chunkDownloaded(t syncTask, err error) {
	if err == nil {
		s.cond.Signal()
	}
}

func (s *cacheReader) chunkReadFailed(segment interfaces.Segment, err error) {
	s.log.Error(fmt.Sprintf("failed to read cache %d-%d: %v", segment.Bound[0], segment.Bound[1], err))

	cache := s.chain.GetSyncCache()
	_ = cache.Delete(segment)
	s.downloader.download(&syncTask{
		from:     segment.Bound[0],
		to:       segment.Bound[1],
		prevHash: segment.PrevHash,
		endHash:  segment.Hash,
	}, true)
}

func (s *cacheReader) clean() {
	cache := s.chain.GetSyncCache()
	cs := cache.Chunks()
	for _, c := range cs {
		_ = cache.Delete(c)
	}

	s.reset()
}

func (s *cacheReader) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.readHeight = 0
	s.buffer = s.buffer[:0]
	s.index = 0

	s.cond.Signal()
}

func (s *cacheReader) removeUselessChunks(cleanWrongEnd bool) {
	height := s.getHeight()
	cache := s.chain.GetSyncCache()
	cs := cache.Chunks()

	for _, c := range cs {
		if c.Bound[1] > height {
			if cleanWrongEnd {
				if c.Bound[1]%syncTaskSize != 0 {
					_ = cache.Delete(c)
				}
			}
			break
		} else {
			_ = cache.Delete(c)
		}
	}
}

func (s *cacheReader) read(c interfaces.Segment) (chunk *Chunk, err error) {
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
		if false == s.running {
			_ = reader.Close()
			return nil, errReaderStopped
		}

		ab, sb, err = reader.Read()
		if err != nil {
			break
		} else if ab != nil {
			if verified == false {
				if err = s.verifier.VerifyNetAb(ab); err != nil {
					break
				}
			}

			if err = chunk.addAccountBlock(ab); err != nil {
				break
			}

		} else if sb != nil {
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

func (s *cacheReader) readLoop() {
	defer s.wg.Done()

	<-s.initReadDone

	cache := s.chain.GetSyncCache()

	var chunk *Chunk
	var err error
	var cs interfaces.SegmentList

Loop:
	for {
		s.removeUselessChunks(false)

		for {
			if false == s.running {
				break Loop
			}

			if cs = cache.Chunks(); len(cs) == 0 {
				s.mu.Lock()
				s.cond.Wait()
				s.mu.Unlock()
				continue
			} else {
				break
			}
		}

		// read chunks
		for _, c := range cs {
			// chunk has read
			if c.Bound[1] <= s.readHeight {
				continue
			}

			// Only read chunks satisfy (chunk[0] == current.Height + 1).
			// Cannot use condition (chunk[0] < current.Height + syncTaskSize).
			// Imagine the following chunks:
			//  current.Height : 10000 ----> missing ----> 10801-14400
			// s.readTo will be set to 14400 if read 10801-14400 successfully (cause 10801 < 1000 + syncTaskSize).
			// The missing chunk 1001-10800 will not be read after download.
			//
			// Chunk is too high, maybe two reasons:
			// 1. chain haven`t grow to c[0]-1, wait for chain grow
			// 2. missing chunks between chain and c, wait for chunk downloaded

			// missing chunk
			if c.Bound[0] > s.readHeight+1 {
				time.Sleep(200 * time.Millisecond)
				// chunk downloaded
				continue Loop
			}

			s.log.Info(fmt.Sprintf("begin read cache %d-%d", c.Bound[0], c.Bound[1]))
			chunk, err = s.read(c)

			// read chunk error
			if err != nil {
				s.chunkReadFailed(c, err)
			} else {
				// will be block
				s.addChunkToBuffer(chunk)
				s.log.Info(fmt.Sprintf("read cache %d-%d done", c.Bound[0], c.Bound[1]))
				// set readTo should be very seriously
				s.readHeight = c.Bound[1]
				s.readHash = c.Hash
			}
		}

		time.Sleep(200 * time.Millisecond)
	}
}
