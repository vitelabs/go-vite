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
	if state == Syncing {
		s.start()
		return
	}

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

	for {
		select {
		case <-s.term:
			atomic.StoreInt32(&s.running, 0)
			return
		default:

		}

		cs := cache.Chunks()

		if len(cs) == 0 {
			time.Sleep(3 * time.Second)
			continue
		}

		var err error
		var reader interfaces.ReadCloser

		height := s.chain.GetLatestSnapshotBlock().Height
		if oldHeight != height {
			oldHeight = height
			lastTime = time.Now()
		}

		for _, c := range cs {
			if c[1] < height {
				_ = cache.Delete(c)
				continue
			}

			if c[0] > height+syncTaskSize {
				if time.Now().Sub(lastTime) > time.Minute {
					s.downloader.download(oldHeight+1, c[0]-1)
				}

				break
			}

			reader, err = cache.NewReader(c[0], c[1])
			if err != nil {
				_ = cache.Delete(c)
				s.downloader.download(c[0], c[1])
				continue
			}

			for {
				var ab *ledger.AccountBlock
				var sb *ledger.SnapshotBlock
				ab, sb, err = reader.Read()
				if err != nil {
					if err == io.EOF {
						err = nil
					}
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

			if err != nil {
				_ = cache.Delete(c)
				s.downloader.download(c[0], c[1])
			}
		}
	}
}
