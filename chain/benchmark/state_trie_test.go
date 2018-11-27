package chain_benchmark

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/trie"
	"testing"
)

func Benchmark_StateTrie(b *testing.B) {
	chainInstance := newTestChainInstance()

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	const (
		TRAVERSE_SNAPSHOT_HEIGHT  = uint64(300000)
		PRINT_PER_NODE_COUNT      = 10 * 10000
		PRINT_PER_SNAPSHOT_HEIGHT = 1000
	)
	tps := newTps(tpsOption{
		name:          "NodeIterator",
		printPerCount: PRINT_PER_NODE_COUNT,
	})

	tps.Start()

	beginHeight := uint64(1)
	if TRAVERSE_SNAPSHOT_HEIGHT > 0 &&
		latestSnapshotBlock.Height > TRAVERSE_SNAPSHOT_HEIGHT {
		beginHeight = latestSnapshotBlock.Height - TRAVERSE_SNAPSHOT_HEIGHT
	}
	totalHashSet := make(map[types.Hash]struct{})
	inHashSet := func(node *trie.TrieNode) bool {
		if _, ok := totalHashSet[*node.Hash()]; ok {
			return false
		}
		return true
	}
	pool := trie.NewCustomTrieNodePool(50*10000, 25*10000)
	for i := beginHeight; i <= latestSnapshotBlock.Height; i++ {
		if i%PRINT_PER_SNAPSHOT_HEIGHT == 0 {
			fmt.Printf("snapshot height: %d\n", i)
		}
		sb, err := chainInstance.GetSnapshotBlockByHeight(i)
		if err != nil {
			b.Fatal(err)
		}

		stateTrie := trie.NewTrie(chainInstance.ChainDb().Db(), &sb.StateHash, pool)
		if stateTrie == nil {
			b.Fatal("stateTrie is nil")
		}
		nodeIterator := stateTrie.NewNodeIterator()

		for nodeIterator.Next(inHashSet) {
			hash := *nodeIterator.Node().Hash()
			totalHashSet[hash] = struct{}{}
			tps.doOne()
		}

	}
	fmt.Printf("hashSet length is %d\n", len(totalHashSet))
	tps.Stop()
	tps.Print()
}
