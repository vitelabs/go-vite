package pending

import (
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"sort"
	"time"
)

type SnapshotchainPool struct {
	cache SnapshotBlockList
}

type SnapshotBlockList []*ledger.SnapshotBlock

func NewSnapshotchainPool(processFunc func(*ledger.SnapshotBlock) bool) *SnapshotchainPool {
	pool := SnapshotchainPool{}

	go func() {
		log.Println("SnapshotchainPool: Start process block")
		turnInterval := time.Duration(2000)
		for {
			if len(pool.cache) <= 0 {
				time.Sleep(turnInterval * time.Millisecond)
				continue
			}

			if processFunc(pool.cache[0]) {
				log.Println("SnapshotchainPool: block process finished.")
				if len(pool.cache) > 0 {
					pool.cache = pool.cache[1:]
				}
			} else {
				log.Println("SnapshotchainPool: block process unsuccess, wait next.")
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

func (pool *SnapshotchainPool) MaxBlock() *ledger.SnapshotBlock {
	block := pool.cache[len(pool.cache)-1]
	return block
}

func (a *SnapshotchainPool) Clear() {
	a.cache = SnapshotBlockList{}
}
func (a *SnapshotchainPool) Add(blocks []*ledger.SnapshotBlock) {
	a.cache = append(a.cache, blocks...)
	a.cache.Sort()
}
