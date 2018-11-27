package chain_benchmark

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

type getSbListByHashParam struct {
	hash    types.Hash
	count   uint64
	forward bool
}

func Benchmark_GetSnapshotBlocksByHash(b *testing.B) {
	chainInstance := newChainInstance("insertAccountBlock", false)

	const (
		PARAMS_LENGTH   = 10 * 10000
		QUERY_NUM_LIMIT = 10 * 100

		FORWARD_TRUE_PROBABILITY = 50
		MAX_COUNT                = 500000
		MIN_COUNT                = 100
	)
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	if latestSnapshotBlock.Height < MIN_COUNT {
		b.Fatal(fmt.Printf("latestSnapshotBlock.Height is less than MIN_COUNT, latestSnapshotBlock.Height is %d, MIN_COUNT is %d", latestSnapshotBlock.Height, MIN_COUNT))
	}

	//var params [PARAMS_LENGTH]*getSbListByHashParam
	for i := 0; i < PARAMS_LENGTH; i++ {

		//count :=
		//params[i] = &getSbListByHashParam{}
	}

	for i := 0; i < QUERY_NUM_LIMIT; i++ {

	}
	//chainInstance.GetSnapshotBlocksByHash()

}
