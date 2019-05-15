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
	chunks() interfaces.SegmentList
	queue() [][2]*ledger.HashHeight
	clean()
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

	running bool
	mu      sync.Mutex
	cond    *sync.Cond

	readHeight uint64

	readable int32

	buffer Chunks
	index  int

	downloadRecord map[interfaces.Segment]peerId

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
		chain:          chain,
		verifier:       verifier,
		downloader:     downloader,
		buffer:         make(Chunks, 0, maxQueueLength),
		log:            netLog.New("module", "cache"),
		readable:       1,
		downloadRecord: make(map[interfaces.Segment]peerId),
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

	cs := s.chunks()
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

	// chunk has been read to buffer
	if segment.Bound[1] <= s.readHeight {
		s.mu.Lock()
		for i, c := range s.buffer {
			if c.SnapshotRange[1].Height == segment.Bound[1] {
				s.buffer = s.buffer[:i]
				s.index = i

				// signal reader, maybe blocked when addChunkToBuffer
				s.cond.Signal()

				if len(s.buffer) > 0 {
					last := s.buffer[len(s.buffer)-1]
					s.readHeight = last.SnapshotRange[1].Height
				} else {
					s.readHeight = t.Bound[0] - 1
				}

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
		s.mu.Lock()
		s.downloadRecord[t.Segment] = t.source
		s.mu.Unlock()
		s.cond.Signal()
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
	s.mu.Lock()
	s.readHeight = 0
	s.buffer = s.buffer[:0]
	s.index = 0
	s.mu.Unlock()

	s.cond.Signal()
}

func (s *cacheReader) clean() {
	cache := s.chain.GetSyncCache()
	for _, c := range cache.Chunks() {
		_ = cache.Delete(c)
	}
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

	var chunk *Chunk
	var fatal bool
	var err error
	var cs interfaces.SegmentList

Loop:
	for {
		for {
			if false == s.running {
				break Loop
			}

			if cs = cache.Chunks(); len(cs) == 0 {
				s.mu.Lock()
				s.cond.Wait()
				s.mu.Unlock()
				continue
			} else if s.canRead() == false {
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

			// missing chunk
			if c.Bound[0] > s.readHeight+1 {
				time.Sleep(200 * time.Millisecond)
				// chunk downloaded
				continue Loop
			}

			s.log.Info(fmt.Sprintf("begin read cache %d-%d", c.Bound[0], c.Bound[1]))
			chunk, fatal, err = s.read(c)

			// read chunk error
			if err != nil {
				s.log.Error(fmt.Sprintf("failed to read cache %d-%d: %v", c.Bound[0], c.Bound[1], err))
				s.chunkReadFailed(c, fatal)
			} else {
				// will be block
				s.addChunkToBuffer(chunk)
				s.log.Info(fmt.Sprintf("read cache %d-%d done", c.Bound[0], c.Bound[1]))
				// set readTo should be very seriously
				s.readHeight = c.Bound[1]
			}
		}

		time.Sleep(200 * time.Millisecond)
	}
}
