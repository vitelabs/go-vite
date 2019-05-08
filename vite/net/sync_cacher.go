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

type syncCacheReader interface {
	ChunkReader
	start()
	stop()
	clean() // clean cache and reset
	reset() // reset state (readTo and requestTo)
	cacheHeight() uint64
	chunks() interfaces.SegmentList
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

type chunkBuffer struct {
	chunks []*Chunk
	index  int
}

func newChunkBuffer(max int) *chunkBuffer {
	return &chunkBuffer{
		chunks: make([]*Chunk, 0, max),
	}
}

func (cb *chunkBuffer) Peek() (c *Chunk) {
	if cb.index < len(cb.chunks) {
		c = cb.chunks[cb.index]
		cb.index++
	}

	return
}

func (cb *chunkBuffer) size() int {
	return len(cb.chunks)
}

func (cb *chunkBuffer) Pop(endHash types.Hash) {
	if len(cb.chunks) > 0 {
		if cb.chunks[0].SnapshotRange[1].Hash == endHash {
			cb.chunks = cb.chunks[1:]
			cb.index--
		}
	}

	return
}

func (cb *chunkBuffer) add(c *Chunk) {
	cb.chunks = append(cb.chunks, c)
}

func (cb *chunkBuffer) reset() {
	cb.chunks = cb.chunks[:0]
	cb.index = 0
}

type cacheReader struct {
	chain      syncChain
	verifier   Verifier
	downloader syncDownloader

	buffer *chunkBuffer

	running   bool
	mu        sync.Mutex
	cond      *sync.Cond
	readTo    uint64
	requestTo uint64
	wg        sync.WaitGroup
	log       log15.Logger
}

func (s *cacheReader) Peek() (c *Chunk) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buffer.Peek()
}

func (s *cacheReader) Pop(endHash types.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer.Pop(endHash)

	s.cond.Signal()
}

func (s *cacheReader) addChunkToBuffer(c *Chunk) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		if false == s.running {
			return
		}

		if s.buffer.size() == 5 {
			s.cond.Wait()
		} else {
			break
		}
	}

	s.buffer.add(c)
}

func newCacheReader(chain syncChain, verifier Verifier, downloader syncDownloader) syncCacheReader {
	s := &cacheReader{
		chain:      chain,
		verifier:   verifier,
		downloader: downloader,
		buffer:     newChunkBuffer(5),
		log:        netLog.New("module", "cache"),
	}

	s.cond = sync.NewCond(&s.mu)

	downloader.addListener(s.chunkDownloaded)

	return s
}

/*
func (s *cacheReader) subSyncState(state SyncState) {
	s.syncState = state

	// sync state maybe change from SyncDone to Syncing.
	// like long time offline, and back online, find higher peers, start sync.
	// because SyncDone cause cacheReader stop, so it need start again.
	if state == Syncing {
		s.start()
		return
	}

	// won`t stop when SyncError, because cache maybe haven`t read done.
	if state == SyncDone || state == SyncCancel {
		s.stop()
		return
	}
}
*/

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

func (s *cacheReader) chunks() interfaces.SegmentList {
	return s.chain.GetSyncCache().Chunks()
}

func (s *cacheReader) chunkDownloaded(from, to uint64, err error) {
	if err == nil {
		s.cond.Signal()
	}
}

func (s *cacheReader) chunkReadFailed(segment interfaces.Segment, err error) {
	s.log.Error(fmt.Sprintf("failed to read cache %d-%d: %v", segment.Bound[0], segment.Bound[1], err))

	cache := s.chain.GetSyncCache()
	_ = cache.Delete(segment)
	s.downloader.download(segment.Bound[0], segment.Bound[1], true)
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

	s.readTo = 0
	s.requestTo = 0

	s.cond.Signal()
}

func (s *cacheReader) cacheHeight() uint64 {
	cache := s.chain.GetSyncCache()
	cs := cache.Chunks()

	if len(cs) > 0 {
		return cs[len(cs)-1].Bound[1]
	}

	return 0
}

func (s *cacheReader) removeUselessChunks() {
	height := s.chain.GetLatestSnapshotBlock().Height
	cache := s.chain.GetSyncCache()
	cs := cache.Chunks()

	for _, c := range cs {
		if c.Bound[1] > height {
			break
		} else {
			_ = cache.Delete(c)
		}
	}
}

func (s *cacheReader) downloadMissingChunks() {
	height := s.chain.GetLatestSnapshotBlock().Height

	cache := s.chain.GetSyncCache()
	cs := cache.Chunks()
	// chunks maybe deleted
	if len(cs) == 0 {
		return
	}

	// height < readTo < requestTo < cacheTo

	if s.readTo < height {
		s.readTo = height
	}

	if s.requestTo < s.readTo {
		s.requestTo = s.readTo
	}

	cacheTo := cs[len(cs)-1].Bound[1]
	if s.requestTo < cacheTo {
		go func(chunks interfaces.SegmentList, from, to uint64) {
			mis := missingSegments(chunks, from, to)
			for _, chunk := range mis {
				if s.downloader.download(chunk[0], chunk[1], false) {
					continue
				} else {
					s.log.Warn(fmt.Sprintf("failed to download %d-%d", chunk[0], chunk[1]))
					break
				}
			}
		}(cs, s.requestTo+1, cacheTo)

		s.requestTo = cacheTo
	}
}

func (s *cacheReader) read(c interfaces.Segment) (chunk *Chunk, err error) {
	cache := s.chain.GetSyncCache()
	reader, err := cache.NewReader(c)
	if err != nil {
		return
	}

	chunk = newChunk(c.PrevHash, c.Bound[0]-1, c.Hash, c.Bound[1], types.RemoteSync)

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
			if err = s.verifier.VerifyNetAb(ab); err != nil {
				break
			}

			if err = chunk.addAccountBlock(ab); err != nil {
				break
			}

		} else if sb != nil {
			if err = s.verifier.VerifyNetSb(sb); err != nil {
				break
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

	return
}

func (s *cacheReader) readLoop() {
	defer s.wg.Done()

	cache := s.chain.GetSyncCache()

	var chunk *Chunk
	var err error
	var cs interfaces.SegmentList

Loop:
	for {
		s.removeUselessChunks()

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

		// request
		s.downloadMissingChunks()

		// read chunks
		for _, c := range cs {
			// chunk has read
			if c.Bound[1] <= s.readTo {
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
			if c.Bound[0] > s.readTo+1 {
				time.Sleep(200 * time.Millisecond)
				// chunk downloaded
				continue Loop
			}

			chunk, err = s.read(c)

			// read chunk error
			if err != nil {
				s.chunkReadFailed(c, err)
			} else {
				// will be block
				s.addChunkToBuffer(chunk)
				// set readTo should be very seriously
				s.readTo = c.Bound[1]
			}
		}
	}
}
