package net

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

type syncCacheReader interface {
	subSyncState(state SyncState)
	start()
	stop()
}

type cacheReader struct {
	chain interface {
		syncCacher
		GetLatestSnapshotBlock() *ledger.SnapshotBlock
	}
	receiver   blockReceiver
	downloader syncDownloader
	syncState  SyncState
	running    int32
	term       chan struct{}
}

func newCacheReader(chain interface {
	syncCacher
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
}, receiver blockReceiver, downloader syncDownloader) syncCacheReader {
	return &cacheReader{
		chain:      chain,
		receiver:   receiver,
		downloader: downloader,
	}
}

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

func (s *cacheReader) start() {
	if atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		s.term = make(chan struct{})
		go s.readCacheLoop()
	}
}

func (s *cacheReader) stop() {
	if atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		close(s.term)
	}
}

func (s *cacheReader) readCacheLoop() {
	cache := s.chain.GetSyncCache()
	var oldHeight uint64
	var lastTime time.Time

	var initDuration = 3 * time.Second
	var maxDuration = 24 * time.Second
	var duration = initDuration
	var timer = time.NewTimer(duration)
	defer timer.Stop()

	var err error
	var reader interfaces.ReadCloser

	for {
		cs := cache.Chunks()

		if len(cs) == 0 {
			// sync error, and no cache to read, then stop.
			if s.syncState == SyncError {
				s.stop()
				return
			}

			select {
			case <-s.term:
				return
			case <-timer.C:

				if duration < maxDuration {
					duration *= 2
				} else {
					duration = initDuration
				}

				timer.Reset(duration)
				continue
			}
		}

		height := s.chain.GetLatestSnapshotBlock().Height
		if oldHeight != height {
			oldHeight = height
			lastTime = time.Now()
		}

		for _, c := range cs {
			// chunk is useless
			if c[1] < height {
				_ = cache.Delete(c)
				continue
			}

			// chunk is too high
			if c[0] > height+syncTaskSize {
				// chain get stuck for a long time, download missing chunks
				if time.Now().Sub(lastTime) > time.Minute {
					s.downloader.download(oldHeight+1, c[0]-1)
				}

				break
			}

			reader, err = cache.NewReader(c[0], c[1])
			if err != nil {
				// chunk is broken, download again
				_ = cache.Delete(c)
				s.downloader.download(c[0], c[1])
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
				_ = cache.Delete(c)
				s.downloader.download(c[0], c[1])
			}
		}
	}
}
