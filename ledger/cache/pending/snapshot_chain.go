package pending

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sort"
	"sync"
	"time"
)

var snapshotchainLog = log15.New("module", "ledger/cache/pending/snapshot_chain")

const tryLimit = 3
const (
	DISCARD = iota
	TRY_AGAIN
	NO_DISCARD
)

type SnapshotchainPool struct {
	cache   SnapshotBlockList
	lock    sync.Mutex
	tryTime int
}

type SnapshotBlockList []*ledger.SnapshotBlock

func NewSnapshotchainPool(processFunc func(*ledger.SnapshotBlock) int) *SnapshotchainPool {
	pool := SnapshotchainPool{}

	go func() {
		turnInterval := time.Duration(2000)
		for {
			pool.lock.Lock()
			if len(pool.cache) <= 0 {
				pool.lock.Unlock()
				time.Sleep(turnInterval * time.Millisecond)
				continue
			}
			block := pool.cache[0]

			if pool.tryTime >= tryLimit {
				pool.lock.Unlock()
				pool.ClearHead()
				continue
			}

			pool.lock.Unlock()

			snapshotchainLog.Info("SnapshotchainPool: Start process block")
			result := processFunc(block)
			if result == DISCARD {
				pool.ClearHead()
			} else {
				if result == TRY_AGAIN {
					pool.lock.Lock()
					pool.tryTime++
					pool.lock.Unlock()
				}

				time.Sleep(turnInterval * time.Millisecond)
			}

		}
	}()

	return &pool
}

func (a SnapshotBlockList) Len() int           { return len(a) }
func (a SnapshotBlockList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SnapshotBlockList) Less(i, j int) bool { return a[i].Height.Cmp(a[j].Height) < 0 }
func (a SnapshotBlockList) Sort() {
	sort.Sort(a)
}

func (pool *SnapshotchainPool) ClearHead() {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	pool.tryTime = 0
	if len(pool.cache) > 0 {
		pool.cache = pool.cache[1:]
	}
}

func (pool *SnapshotchainPool) Add(blocks []*ledger.SnapshotBlock) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if blocks == nil || len(blocks) == 0 {
		return
	}

	pool.cache = append(pool.cache, blocks...)
	pool.cache.Sort()
}
