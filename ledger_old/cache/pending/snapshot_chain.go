package pending

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
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
	cache   sbCacheType
	lock    sync.Mutex
	tryTime int
}
type sbCacheType []sbCacheItem

type sbCacheItem struct {
	block *ledger.SnapshotBlock
	peer  *protoTypes.Peer
	id    uint64
}

type sbProcessInterface interface {
	ProcessBlock(*ledger.SnapshotBlock, *protoTypes.Peer, uint64) int
}

type SnapshotBlockList []*ledger.SnapshotBlock

func NewSnapshotchainPool(processor sbProcessInterface) *SnapshotchainPool {
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
			cacheItem := pool.cache[0]

			if pool.tryTime >= tryLimit {
				pool.lock.Unlock()
				pool.ClearHead()
				continue
			}

			pool.lock.Unlock()

			snapshotchainLog.Info("SnapshotchainPool: Start process block")
			result := processor.ProcessBlock(cacheItem.block, cacheItem.peer, cacheItem.id)
			if result == DISCARD {
				pool.ClearHead()
			} else {
				if result == TRY_AGAIN {
					pool.lock.Lock()
					pool.tryTime++
					pool.lock.Unlock()
				}

				time.Sleep(2000 * time.Millisecond)
			}

		}
	}()

	return &pool
}

func (a sbCacheType) Len() int           { return len(a) }
func (a sbCacheType) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sbCacheType) Less(i, j int) bool { return a[i].block.Height.Cmp(a[j].block.Height) < 0 }
func (a sbCacheType) Sort() {
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

func (pool *SnapshotchainPool) Add(blocks []*ledger.SnapshotBlock, peer *protoTypes.Peer, id uint64) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if blocks == nil || len(blocks) == 0 {
		return
	}

	for _, block := range blocks {
		pool.cache = append(pool.cache, sbCacheItem{
			block: block,
			peer:  peer,
			id:    id,
		})
	}

	pool.cache.Sort()
}
