package chain_benchmark

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"math/rand"
	"testing"
)

type getSbListByHashParam struct {
	hash    types.Hash
	count   uint64
	forward bool
}

type getSbListByHeight struct {
	height  uint64
	count   uint64
	forward bool
}

func Benchmark_GetSnapshotBlocksByHash(b *testing.B) {
	//chainInstance := newChainInstance("insertAccountBlock", false)
	chainInstance := newTestChainInstance()

	const (
		PARAMS_LENGTH   = 2 * 10000
		QUERY_NUM_LIMIT = 10 * 10000

		CONTAIN_SNAPSHOT_CONTENT = false
		FORWARD_TRUE_PROBABILITY = 50
		MAX_COUNT                = 200
		MIN_COUNT                = 100

		PREPARE_PARAMS_PRINT_PER_COUNT = 1000
		PRINT_PER_COUNT                = 100 * 10000
		PRINT_PER_QUERY_TIME           = 100 * 1000
	)
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	if latestSnapshotBlock.Height < MIN_COUNT {
		b.Fatal(fmt.Printf("latestSnapshotBlock.Height is less than MIN_COUNT, latestSnapshotBlock.Height is %d, MIN_COUNT is %d", latestSnapshotBlock.Height, MIN_COUNT))
	}

	var params [PARAMS_LENGTH]*getSbListByHashParam
	for i := 0; i < PARAMS_LENGTH; i++ {

		if i%PREPARE_PARAMS_PRINT_PER_COUNT == 0 {
			fmt.Printf("prepare %d params\n", i)
		}
		randomHeight := rand.Uint64() % latestSnapshotBlock.Height
		if randomHeight <= 0 {
			randomHeight = 1
		}
		sb, err := chainInstance.GetSnapshotBlockByHeight(randomHeight)
		if err != nil {
			b.Fatal(err)
		}

		count := MIN_COUNT + rand.Uint64()%(MAX_COUNT-MIN_COUNT)
		forward := false
		if rand.Intn(100) < FORWARD_TRUE_PROBABILITY {
			forward = true
		}

		params[i] = &getSbListByHashParam{
			hash:    sb.Hash,
			count:   count,
			forward: forward,
		}
	}
	fmt.Println(fmt.Sprintf("latest snapshot block height is %d", latestSnapshotBlock.Height))

	tps := newTps(tpsOption{
		name:          "getAccountBlocksByHash|blockNum",
		printPerCount: PRINT_PER_COUNT,
	})

	tps2 := newTps(tpsOption{
		name:          "getAccountBlocksByHash|queryTimes",
		printPerCount: PRINT_PER_QUERY_TIME,
	})
	tps.Start()
	tps2.Start()

	for i := 0; i < QUERY_NUM_LIMIT; i++ {
		randomParamIndex := rand.Intn(PARAMS_LENGTH)
		param := params[randomParamIndex]
		sbs, err := chainInstance.GetSnapshotBlocksByHash(&param.hash, param.count, param.forward, CONTAIN_SNAPSHOT_CONTENT)
		if err != nil {
			b.Fatal(err)
		}
		tps.do(uint64(len(sbs)))
		tps2.doOne()
	}

	tps.Stop()
	tps.Print()

	tps2.Stop()
	tps2.Print()
}

func Benchmark_GetSnapshotBlocksByHeight(b *testing.B) {
	chainInstance := newTestChainInstance()

	const (
		PARAMS_LENGTH   = 2 * 10000
		QUERY_NUM_LIMIT = 100 * 10000

		CONTAIN_SNAPSHOT_CONTENT = false
		FORWARD_TRUE_PROBABILITY = 50
		MAX_COUNT                = uint64(200)
		SUGGEST_MIN_COUNT        = uint64(100)

		PRINT_PER_COUNT      = 100 * 10000
		PRINT_PER_QUERY_TIME = 1 * 100
	)
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	var params [PARAMS_LENGTH]*getSbListByHeight
	for i := 0; i < PARAMS_LENGTH; i++ {
		randomHeight := rand.Uint64() % latestSnapshotBlock.Height
		if randomHeight <= 0 {
			randomHeight = 1
		}

		forward := false

		if rand.Intn(100) < FORWARD_TRUE_PROBABILITY {
			forward = true
		}

		count := SUGGEST_MIN_COUNT + rand.Uint64()%(MAX_COUNT-SUGGEST_MIN_COUNT)

		if forward {
			targetHeight := randomHeight + count
			if targetHeight > latestSnapshotBlock.Height {
				targetHeight = latestSnapshotBlock.Height
			}
			count = targetHeight - randomHeight
		} else if randomHeight < count {
			count = randomHeight
		}

		params[i] = &getSbListByHeight{
			height:  randomHeight,
			forward: forward,
			count:   count,
		}
	}
	tps := newTps(tpsOption{
		name:          "getAccountBlocksByHeight|blockNum",
		printPerCount: PRINT_PER_COUNT,
	})

	tps2 := newTps(tpsOption{
		name:          "getAccountBlocksByHeight|queryTimes",
		printPerCount: PRINT_PER_QUERY_TIME,
	})
	tps.Start()
	tps2.Start()

	for i := 0; i < QUERY_NUM_LIMIT; i++ {
		randomParamIndex := rand.Intn(PARAMS_LENGTH)
		param := params[randomParamIndex]
		sbs, err := chainInstance.GetSnapshotBlocksByHeight(param.height, param.count, param.forward, CONTAIN_SNAPSHOT_CONTENT)
		if err != nil {
			b.Fatal(err)
		}
		tps.do(uint64(len(sbs)))
		tps2.doOne()
	}

	tps.Stop()
	tps.Print()

	tps2.Stop()
	tps2.Print()
}

func Benchmark_GetSnapshotBlockByHash(b *testing.B) {
	chainInstance := newTestChainInstance()

	const (
		QUERY_NUM_LIMIT = 1000 * 10000
		PRINT_PER_COUNT = 50 * 10000
		PARAMS_LENGTH   = 5 * 10000

		ONLY_HEAD = true
	)

	if ONLY_HEAD {
		fmt.Println("only head")
	}

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	var params [PARAMS_LENGTH]types.Hash
	for i := 0; i < PARAMS_LENGTH; i++ {
		randomHeight := rand.Uint64() % latestSnapshotBlock.Height
		if randomHeight <= 0 {
			randomHeight = 1
		}
		sb, err := chainInstance.GetSnapshotBlockByHeight(randomHeight)
		if err != nil {
			b.Fatal(err)
		}

		params[i] = sb.Hash

	}

	fmt.Println(fmt.Sprintf("latest snapshot block height is %d", latestSnapshotBlock.Height))

	tps := newTps(tpsOption{
		name:          "getAccountBlockByHash",
		printPerCount: PRINT_PER_COUNT,
	})

	tps.Start()

	for i := 0; i < QUERY_NUM_LIMIT; i++ {
		randomParamIndex := rand.Intn(PARAMS_LENGTH)
		param := params[randomParamIndex]

		var err error
		if ONLY_HEAD {
			_, err = chainInstance.GetSnapshotBlockHeadByHash(&param)
		} else {
			_, err = chainInstance.GetSnapshotBlockByHash(&param)
		}

		if err != nil {
			b.Fatal(err)
		}

		tps.doOne()
	}

	tps.Print()
	tps.Stop()
}

func Benchmark_GetSnapshotBlockByHeight(b *testing.B) {
	chainInstance := newTestChainInstance()

	const (
		QUERY_NUM_LIMIT = 1000 * 10000
		PRINT_PER_COUNT = 50 * 10000
		ONLY_HEAD       = true
	)

	if ONLY_HEAD {
		fmt.Println("only head")
	}

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	fmt.Println(fmt.Sprintf("latest snapshot block height is %d", latestSnapshotBlock.Height))
	tps := newTps(tpsOption{
		name:          "getAccountBlockByHeight",
		printPerCount: PRINT_PER_COUNT,
	})

	tps.Start()

	for i := 0; i < QUERY_NUM_LIMIT; i++ {
		randomHeight := rand.Uint64() % latestSnapshotBlock.Height

		var err error
		if ONLY_HEAD {
			_, err = chainInstance.GetSnapshotBlockHeadByHeight(randomHeight)
		} else {
			_, err = chainInstance.GetSnapshotBlockByHeight(randomHeight)
		}
		if err != nil {
			b.Fatal(err)
		}

		tps.doOne()
	}

	tps.Print()
	tps.Stop()
}
