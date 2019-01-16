package chain_benchmark

import (
	"math/rand"
	"testing"
)

func Benchmark_getGetConfirmSubLedger(b *testing.B) {
	chainInstance := newChainInstance("insertAccountBlock", false)
	const (
		QUERY_NUM_LIMIT               = 10000 * 10000
		SNAPSHOT_COUNT                = 10
		PRINT_PER_SNAPSHOTBLOCK_COUNT = 10
		PRINT_PER_ACCOUNTBLOCK_COUNT  = 10 * 10000
	)

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	tps := newTps(tpsOption{
		name:          "getGetConfirmSubLedger|snapshotBlock",
		printPerCount: PRINT_PER_SNAPSHOTBLOCK_COUNT,
	})

	tps2 := newTps(tpsOption{
		name:          "getGetConfirmSubLedger|accountBlock",
		printPerCount: PRINT_PER_ACCOUNTBLOCK_COUNT,
	})

	tps.Start()
	tps2.Start()

	for i := 0; i < QUERY_NUM_LIMIT; i++ {
		fromHeight := rand.Uint64() % latestSnapshotBlock.Height
		if fromHeight <= 0 {
			fromHeight = uint64(1)
		}
		toHeight := fromHeight + SNAPSHOT_COUNT
		if toHeight > latestSnapshotBlock.Height {
			toHeight = latestSnapshotBlock.Height
		}

		snapshotBlocks, subLedger, err := chainInstance.GetConfirmSubLedger(fromHeight, toHeight)

		if err != nil {
			b.Fatal(err)
		}
		tps.do(uint64(len(snapshotBlocks)))
		for _, blocks := range subLedger {
			tps2.do(uint64(len(blocks)))
		}
	}

	tps.Print()
	tps2.Print()

	tps.Stop()
	tps2.Stop()
}
