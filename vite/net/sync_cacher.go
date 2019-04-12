package net

import (
	"io"
	"sync"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

type syncCacheReader interface {
	start()
	stop()
}

type cacheReader struct {
	chain      syncChain
	receiver   blockReceiver
	downloader syncDownloader
	running    bool
	mu         sync.Mutex
	cond       *sync.Cond
}

func newCacheReader(chain syncChain, receiver blockReceiver, downloader syncDownloader) syncCacheReader {
	s := &cacheReader{
		chain:      chain,
		receiver:   receiver,
		downloader: downloader,
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
	s.running = true
	s.mu.Unlock()

	go s.readLoop()
}

func (s *cacheReader) stop() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	s.cond.Signal()
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

func (s *cacheReader) readLoop() {
	cache := s.chain.GetSyncCache()

	var err error
	var reader interfaces.ReadCloser
	var height uint64
	var chunkReadEnd uint64
	var requestEnd uint64

Loop:
	for {
		cs := cache.Chunks()

		s.mu.Lock()
		for len(cs) == 0 {
			if false == s.running {
				s.mu.Unlock()
				break Loop
			}

			s.cond.Wait()
		}
		s.mu.Unlock()

		height = s.chain.GetLatestSnapshotBlock().Height

		// clean useless chunks
		var i int
		var c [2]uint64
		for i, c = range cs {
			if c[1] < height {
				_ = cache.Delete(c)
			} else {
				break
			}
		}

		cs = cs[i:]
		if len(cs) == 0 {
			continue
		}

		// request missing chunks
		chunkTo := cs[len(cs)-1][1]
		if requestEnd < chunkTo {
			requestEnd = chunkTo

			chunks := make([][2]uint64, len(cs))
			for i, c = range cs {
				chunks[i] = c
			}

			mis := missingChunks(chunks, height+1, requestEnd)
			for _, chunk := range mis {
				s.downloader.download(chunk[0], chunk[1], false)
			}
		}

		// read chunks
		for _, c = range cs {
			if c[1] < chunkReadEnd {
				continue
			}

			// chunk is too high
			if c[0] > height+syncTaskSize {
				// wait for download
				s.mu.Lock()
				s.cond.Wait()
				s.mu.Unlock()
				// chunk downloaded
				continue Loop
			}

			reader, err = cache.NewReader(c[0], c[1])
			if err != nil {
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
				s.handleChunkError(c)
			} else {
				chunkReadEnd = c[1]
			}
		}
	}
}
