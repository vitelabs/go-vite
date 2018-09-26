package pool

import (
	"sort"
	"strconv"
	"testing"

	"github.com/vitelabs/go-vite/ledger"
)

func TestByHeight(t *testing.T) {
	blocks := genCommonBlocks(10)

	sort.Sort(ByHeight(blocks))

	for k, v := range blocks {
		println(strconv.Itoa(k) + "::" + strconv.FormatUint(v.Height(), 10))
	}

	bcPool := BCPool{}

	err := bcPool.checkChain(blocks)
	if err != nil {
		t.Error(err)
	}
}
func genCommonBlocks(n int) []commonBlock {
	var results []commonBlock
	for i := 0; i < n; i++ {
		block := &ledger.SnapshotBlock{Height: uint64(i)}
		results = append(results, newSnapshotPoolBlock(block, &ForkVersion{}))
	}

	for i := n*2 - 1; i >= n; i-- {
		block := &ledger.SnapshotBlock{Height: uint64(i)}
		results = append(results, newSnapshotPoolBlock(block, &ForkVersion{}))
	}
	return results
}
