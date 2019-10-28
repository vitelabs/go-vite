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

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

var errReaderStopped = errors.New("cache reader stopped")
var errReaderInterrupt = errors.New("cache reader interrupt")

const maxQueueSize = 10 << 20 // 10MB
const maxQueueLength = 5

type syncCacheReader interface {
	ChunkReader
	start()
	stop()
	compareCache(start *ledger.HashHeight, hhs []*HashHeightPoint) syncTasks
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

	downloadRecord map[string]peerId

	blackBlocks map[types.Hash]struct{}

	wg  sync.WaitGroup
	log log15.Logger
}

func (s *cacheReader) cache(from, to uint64) (ret interfaces.SegmentList) {
	cs := s.localChunks()
	return retrieveSegments(cs, from, to)
}

func retrieveSegments(cs interfaces.SegmentList, from, to uint64) (ret interfaces.SegmentList) {
	if len(cs) == 0 {
		return nil
	}

	index := sort.Search(len(cs), func(i int) bool {
		return cs[i].To >= from
	})

	for i := index; i < len(cs); i++ {
		if cs[i].From > to {
			break
		}
		ret = append(ret, cs[i])
	}

	return
}

func (s *cacheReader) caches() interfaces.SegmentList {
	cs := s.localChunks()
	return mergeSegments(cs)
}

func mergeSegments(cs interfaces.SegmentList) (cs2 interfaces.SegmentList) {
	if len(cs) == 0 {
		return
	}

	segment := cs[0]
	for i := 1; i < len(cs); i++ {
		if cs[i].PrevHash == segment.Hash && cs[i].From == segment.To+1 {
			segment.Hash = cs[i].Hash
			segment.To = cs[i].To
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

func newCacheReader(chain syncChain, verifier Verifier, downloader syncDownloader, irreader IrreversibleReader, blackBlocks map[types.Hash]struct{}) *cacheReader {
	if len(blackBlocks) == 0 {
		blackBlocks = make(map[types.Hash]struct{})
	}

	s := &cacheReader{
		chain:          chain,
		verifier:       verifier,
		downloader:     downloader,
		irreader:       irreader,
		running:        false,
		mu:             sync.Mutex{},
		cond:           nil,
		readHeight:     0,
		readable:       1,
		buffer:         make(Chunks, 0, maxQueueLength),
		downloadRecord: make(map[string]peerId),
		blackBlocks:    blackBlocks,
		wg:             sync.WaitGroup{},
		log:            netLog.New("module", "cache"),
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

// start is use to confirm the first task.PrevHash
func constructTasks(hhs []*HashHeightPoint) (ts syncTasks) {
	const maxSnapshotChunksOneTask = 2000 // almost 2000 * 0.5k = 1mb
	const maxSizeOneTask = 1 << 20        // 1mb

	var seg = interfaces.Segment{
		From:     hhs[0].Height + 1,
		To:       0,
		Hash:     types.Hash{},
		PrevHash: hhs[0].Hash,
		Points: []*ledger.HashHeight{
			{hhs[0].Height, hhs[0].Hash},
		},
	}
	var size uint64
	var hasChunkInSeg bool

	for i := 1; i < len(hhs); i++ {
		hp := hhs[i]
		size += hp.Size

		if hp.Height-seg.From+1 > maxSnapshotChunksOneTask || size > maxSizeOneTask {
			if !hasChunkInSeg {
				seg.To = hp.Height
				seg.Hash = hp.Hash
				seg.Points = append(seg.Points, &hp.HashHeight) // last point
			}

			ts = append(ts, &syncTask{
				Segment: seg,
			})

			// create a new task
			seg = interfaces.Segment{
				From:     seg.To + 1,
				PrevHash: seg.Hash,
				Points: []*ledger.HashHeight{
					{seg.To, seg.Hash}, // first point
					{hp.Height, hp.Hash},
				},
			}

			size = hp.Size
		}

		hasChunkInSeg = true
		seg.To = hp.Height
		seg.Hash = hp.Hash
		seg.Points = append(seg.Points, &hp.HashHeight)
	}

	seg.To = hhs[len(hhs)-1].Height
	seg.Hash = hhs[len(hhs)-1].Hash

	ts = append(ts, &syncTask{
		Segment: seg,
	})

	return
}

/*
 * eg.
 * HashHeightList is [100, 200, 300, 400 ... 10000]
 * segments is [701-1000, 1201-1500]
 * if segments are right, then tasks will be [construct(100, 700), construct(1000~1200), construct(1500, 10000)]
 */
func compareCache(segments interfaces.SegmentList, hashHeightList []*HashHeightPoint, deleteChunk func(segment interfaces.Segment)) (ts syncTasks) {

	//choose valid segments and delete invalid segments
	var validSegments interfaces.SegmentList
	var index int
	hashHeightStart := hashHeightList[0].Height
	hashHeightEnd := hashHeightList[len(hashHeightList)-1].Height
	for _, segment := range segments {
		// too low
		if segment.To <= hashHeightStart {
			continue
		}

		// too high
		if segment.From > hashHeightEnd {
			break
		}

		var fromOK, toOK bool
		for index < len(hashHeightList) {
			hashHeight := hashHeightList[index]
			if (segment.From == hashHeight.Height+1) && (segment.PrevHash == hashHeight.Hash) {
				fromOK = true
			}

			if (segment.To == hashHeight.Height) && (segment.Hash == hashHeight.Hash) {
				toOK = true
			}

			// hashHeight is out of segment, should not continue
			if hashHeight.Height >= segment.To {
				break
			}

			index++
		}

		if fromOK && toOK {
			validSegments = append(validSegments, segment)
		} else {
			deleteChunk(segment)
		}
	}

	if validSegments == nil {
		ts = constructTasks(hashHeightList)
		return
	} else {

		//merge continuous valid segments
		var continuousSegment interfaces.SegmentList
		for _, segment := range validSegments {
			if len(continuousSegment) == 0 {
				continuousSegment = append(continuousSegment, segment)
			} else {
				lastSegment := continuousSegment[len(continuousSegment)-1]
				if lastSegment.To+1 == segment.From {
					continuousSegment[len(continuousSegment)-1].To = segment.To
				} else {
					continuousSegment = append(continuousSegment, segment)
				}
			}
		}

		//exclude valid segment from hashHeightList
		var points = make(map[uint64]uint64)
		for _, segment := range continuousSegment {
			points[segment.From-1] = segment.To
		}

		var start int
		var segmentEnd uint64
		var inSegment bool
		for index, hashHeight := range hashHeightList {
			if hashHeight.Height >= segmentEnd {
				if inSegment {
					start = index
				}
				inSegment = false
			} else {
				continue
			}

			v, ok := points[hashHeight.Height]
			if !ok {
				continue
			} else {
				// segment: 100 - 400
				// hashlist: 100, 200, 300
				// then start and index are both 0
				if !inSegment {
					if index > 0 {
						ts = append(ts, constructTasks(hashHeightList[start:index+1])...)
					}
					inSegment = true
				}
				segmentEnd = v
			}
		}
		ts = append(ts, constructTasks(hashHeightList[start:])...)
	}

	return
}

//func compareCache(segments interfaces.SegmentList, hashHeightList []*message.HashHeightPoint, deleteChunk func(segment interfaces.Segment)) (ts syncTasks) {
//	var start, from, index int
//	var fromOK, toOK bool
//	var hashHeight *message.HashHeightPoint
//	for _, segment := range segments {
//		fromOK = false
//		toOK = false
//		for index < len(hashHeightList) {
//			hashHeight = hashHeightList[index]
//			// compare from
//			if segment.From == hashHeight.Height+1 {
//				if segment.PrevHash == hashHeight.Hash {
//					fromOK = true
//					from = index
//				}
//				break
//			} else if segment.From < hashHeight.Height {
//				break
//			}
//
//			index++
//		}
//
//		if fromOK {
//			// compare to
//			for index < len(hashHeightList) {
//				hashHeight = hashHeightList[index]
//				if segment.To == hashHeight.Height {
//					if segment.Hash == hashHeight.Hash {
//						// to is ok
//						toOK = true
//					}
//					break
//				} else if segment.To < hashHeight.Height {
//					break
//				}
//
//				index++
//			}
//		}
//
//		if fromOK && toOK {
//			// add tasks
//			ts = append(ts, constructTasks(hashHeightList[start:from+1])...)
//			start = index
//			from = index
//		} else {
//			deleteChunk(segment)
//		}
//	}
//
//	// rest HashHeightList
//	ts = append(ts, constructTasks(hashHeightList[from:])...)
//
//	return
//}

func (s *cacheReader) compareCache(start *ledger.HashHeight, hhs []*HashHeightPoint) syncTasks {
	s.pause()
	defer s.resume() // will signal reader
	atomic.CompareAndSwapUint64(&s.readHeight, 0, start.Height)

	hhs[0].Height = start.Height
	hhs[0].Hash = start.Hash

	cs := s.localChunks()
	if len(cs) == 0 {
		return constructTasks(hhs)
	}

	if cs[0].From >= hhs[len(hhs)-1].Height {
		return constructTasks(hhs)
	}

	last := cs[len(cs)-1]
	if last.To <= start.Height {
		return constructTasks(hhs)
	}

	// chunk and tasks are overlapped
	return compareCache(cs, hhs, s.deleteChunk)
}

func (s *cacheReader) deleteChunk(segment interfaces.Segment) {
	cache := s.chain.GetSyncCache()
	_ = cache.Delete(segment)

	s.log.Warn(fmt.Sprintf("delete chunk %d-%d/%s/%s", segment.From, segment.To, segment.PrevHash, segment.Hash))

	// chunk has been read to buffer
	if segment.To <= s.readHeight {
		s.mu.Lock()
		for i, c := range s.buffer {
			if c.SnapshotRange[1].Height == segment.To {
				s.buffer = s.buffer[:i]

				// todo
				s.readHeight = c.SnapshotRange[0].Height

				s.log.Warn(fmt.Sprintf("delete chunk in buffer from %d", i))

				// signal reader, maybe blocked when addChunkToBuffer
				s.cond.Signal()

				break
			}
		}
		s.mu.Unlock()
	}
}

func (s *cacheReader) localChunks() interfaces.SegmentList {
	return s.chain.GetSyncCache().Chunks()
}

func (s *cacheReader) chunkDownloaded(t syncTask, err error) {
	if err == nil {
		s.mu.Lock()
		s.downloadRecord[t.Segment.String()] = t.source
		s.mu.Unlock()
	}
}

func (s *cacheReader) chunkReadFailed(segment interfaces.Segment, fatal bool) {
	if fatal {
		s.mu.Lock()
		id := s.downloadRecord[segment.String()]
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
	atomic.StoreUint64(&s.readHeight, 0)
	s.buffer = s.buffer[:0]
	s.mu.Unlock()

	cache := s.chain.GetSyncCache()
	cs := cache.Chunks()
	if len(cs) > 0 {
		last := cs[len(cs)-1]
		if last.To%syncTaskSize != 0 {
			_ = cache.Delete(last)
		}
	}

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
		if c.To > height {
			if cleanWrong {
				if c.From%syncTaskSize != 1 && c.From != genesisHeight+1 {
					_ = cache.Delete(c)
				} else if c.To%syncTaskSize != 0 {
					_ = cache.Delete(c)
				}
			} else {
				break
			}
		} else {
			delete(s.downloadRecord, c.String())
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

	chunk = newChunk(c.PrevHash, c.From-1, c.Hash, c.To, types.RemoteSync)

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
				if err = s.verifier.VerifyNetAccountBlock(ab); err != nil {
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
				if err = s.verifier.VerifyNetSnapshotBlock(sb); err != nil {
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

		readHeight := atomic.LoadUint64(&s.readHeight)
		// read chunks
		for _, c := range cs {
			// changed
			currReadHeight := atomic.LoadUint64(&s.readHeight)
			if currReadHeight != readHeight {
				continue Loop
			}

			// chunk has read
			if c.To <= readHeight {
				continue
			}

			// missing chunk
			if c.From > readHeight+1 {
				time.Sleep(200 * time.Millisecond)
				// chunk downloaded
				continue Loop
			}

			s.log.Info(fmt.Sprintf("begin read cache %d-%d", c.From, c.To))
			chunk, fatal, err := s.read(c)

			if err != nil {
				// read chunk error
				s.log.Error(fmt.Sprintf("failed to read cache %d-%d: %v", c.From, c.To, err))
				s.chunkReadFailed(c, fatal)
			} else {
				s.log.Info(fmt.Sprintf("read cache %d-%d done", c.From, c.To))
				if atomic.CompareAndSwapUint64(&s.readHeight, readHeight, c.To) {
					readHeight = c.To
					// will be block
					s.addChunkToBuffer(chunk)
				}
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
			if c.To < irevBlock.Height {
				_ = cache.Delete(c)
			}
		}

		time.Sleep(duration)
	}
}
