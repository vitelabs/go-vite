package pending

import (
	"github.com/vitelabs/go-vite/ledger"
	"sort"
	"time"
)

type SnapshotchainPool []*ledger.SnapshotBlock

func NewSnapshotchainPool (processFunc func(*ledger.SnapshotBlock)bool) *SnapshotchainPool {
	pool := SnapshotchainPool{}

	go func () {
		turnInterval := time.Duration(2000)
		for {
			if len(pool) <= 0 {
				time.Sleep(turnInterval)
			}

			if processFunc(pool[0]) {
				pool = pool[1:]
			} else {
				time.Sleep(turnInterval)
			}
		}
	}()

	return &pool
}

func (a SnapshotchainPool) Len() int {return len(a)}
func (a SnapshotchainPool) Swap(i, j int) {a[i], a[j] = a[j], a[i]}
func (a SnapshotchainPool) Less(i, j int) bool {return a[i].Height.Cmp(a[j].Height) < 0}
func (a SnapshotchainPool) Sort () {
	sort.Sort(a)
}

func (a SnapshotchainPool) Add (blocks []*ledger.SnapshotBlock) {
	a = append(a, blocks...)
	a.Sort()
}