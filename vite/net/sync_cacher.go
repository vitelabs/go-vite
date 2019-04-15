package net

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

type syncCacheReader interface {
	start()
	stop()
	// clean cache and reset state
	clean()
	// reset state
	reset()
	cacheHeight() uint64
}

type cacheReader struct {
	chain      syncChain
	receiver   blockReceiver
	downloader syncDownloader
	running    bool
	mu         sync.Mutex
	cond       *sync.Cond
	readTo     uint64
	requestTo  uint64
	wg         sync.WaitGroup
	log        log15.Logger
}

func newCacheReader(chain syncChain, receiver blockReceiver, downloader syncDownloader) syncCacheReader {
	s := &cacheReader{
		chain:      chain,
		receiver:   receiver,
		downloader: downloader,
		log:        netLog.New("module", "cache"),
	}
	s.cond = sync.NewCond(&s.mu)

	downloader.addListener(s.handleChunkDone)

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

func (s *cacheReader) handleChunkDone(from, to uint64, err error) {
	if err == nil {
		s.cond.Signal()
	}
}

func (s *cacheReader) handleChunkError(chunk [2]uint64) {
	cache := s.chain.GetSyncCache()
	_ = cache.Delete(chunk)
	s.downloader.download(chunk[0], chunk[1], true)
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
		return cs[len(cs)-1][1]
	}

	return 0
}

func (s *cacheReader) removeUselessChunks() {
	height := s.chain.GetLatestSnapshotBlock().Height
	cache := s.chain.GetSyncCache()
	cs := cache.Chunks()

	for _, c := range cs {
		if c[1] > height {
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

	cacheTo := cs[len(cs)-1][1]
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

func (s *cacheReader) readLoop() {
	defer s.wg.Done()

	cache := s.chain.GetSyncCache()

	var err error
	var reader interfaces.ReadCloser
	var height uint64

Loop:
	for {
		s.removeUselessChunks()

		cs := cache.Chunks()

		s.mu.Lock()
		for len(cs) == 0 && s.running {
			s.cond.Wait()
		}
		if false == s.running {
			s.mu.Unlock()
			break Loop
		}
		s.mu.Unlock()

		// request
		s.downloadMissingChunks()

		height = s.chain.GetLatestSnapshotBlock().Height

		// read chunks
		for _, c := range cs {
			// chunk has read
			if c[1] <= s.readTo {
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
			if c[0] > height+1 {
				time.Sleep(200 * time.Millisecond)
				// chunk downloaded
				continue Loop
			}

			reader, err = cache.NewReader(c[0], c[1])
			if err != nil {
				s.log.Error(fmt.Sprintf("failed to read cache %d-%d: %v", c[0], c[1], err))
				s.handleChunkError(c)
				continue
			}

			// read chunk
			for {
				var ab *ledger.AccountBlock
				var sb *ledger.SnapshotBlock
				ab, sb, err = reader.Read()
				if err != nil {
					break
				} else if ab != nil {
					if err = s.receiver.receiveAccountBlock(ab, types.RemoteSync); err != nil {
						break
					}
				} else if sb != nil {
					if err = s.receiver.receiveSnapshotBlock(sb, types.RemoteSync); err != nil {
						break
					}
				}
			}

			// read chunk error
			if err != nil && err != io.EOF {
				s.log.Error(fmt.Sprintf("failed to read cache %d-%d: %v", c[0], c[1], err))
				s.handleChunkError(c)
			} else {
				// set readTo should be very seriously
				s.readTo = c[1]
			}
		}
	}
}
