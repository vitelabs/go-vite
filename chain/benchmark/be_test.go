package chain_benchmark

import (
	"fmt"
	"math/rand"
	"testing"
)

func Benchmark_GetBlockEvent(b *testing.B) {
	chainInstance := newChainInstance("insertAccountBlock", false)

	const (
		QUERY_TIMES     = 1000 * 10000
		PRINT_PER_COUNT = 20 * 10000
	)

	latestEventId, err := chainInstance.GetLatestBlockEventId()
	if err != nil {
		b.Fatal(err.Error())
	}

	tps := newTps(tpsOption{
		name:          "GetEvent",
		printPerCount: PRINT_PER_COUNT,
	})
	tps.Start()
	for i := 0; i < QUERY_TIMES; i++ {
		_, _, err := chainInstance.GetEvent(rand.Uint64() % latestEventId)
		if err != nil {
			b.Fatal(err)
		}
		tps.doOne()
	}
	tps.Stop()
	tps.Print()

	fmt.Printf("lastEventId is %d, query %d times.\n ", latestEventId, QUERY_TIMES)
}
