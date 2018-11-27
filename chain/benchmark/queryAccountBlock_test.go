package chain_benchmark

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"math/rand"
	"testing"
)

type getAbsByHashParam struct {
	addr    types.Address
	origin  *types.Hash
	count   uint64
	forward bool
}

func Benchmark_GetAccountBlocksByHash(b *testing.B) {
	chainInstance := newChainInstance("insertAccountBlock", false)

	const (
		PARAMS_LENGTH = 10 * 10000
		TRY_COUNT     = 50 * 10000

		FORWARD_TRUE_PROBABILITY = 50
		MAX_COUNT                = 500000
		MIN_COUNT                = 1000
	)
	// create getAbsByHashParam
	var params []getAbsByHashParam

	allLatestBlock, _ := chainInstance.GetAllLatestAccountBlock()
	for i := 0; len(params) < PARAMS_LENGTH && i < TRY_COUNT; i++ {
		block := allLatestBlock[rand.Intn(len(allLatestBlock))]
		addr := block.AccountAddress

		if block.Height < MIN_COUNT {
			continue
		}

		randomHeight := rand.Uint64() % block.Height
		if randomHeight <= 0 {
			randomHeight = 1
		}

		randomBlock, _ := chainInstance.GetAccountBlockByHeight(&addr, randomHeight)

		forward := false
		if rand.Intn(100) < FORWARD_TRUE_PROBABILITY {
			forward = true
		}

		count := (rand.Uint64() % (MAX_COUNT - MIN_COUNT)) + MIN_COUNT

		params = append(params, getAbsByHashParam{
			addr:    addr,
			origin:  &randomBlock.Hash,
			count:   count,
			forward: forward,
		})
	}

	fmt.Printf("params length: %d\n", len(params))
	getAccountBlocksByHash(chainInstance, params)
}

func getAccountBlocksByHash(chainInstance chain.Chain, testParams []getAbsByHashParam) {
	const (
		QUERY_NUM_LIMIT = 10000 * 10000

		PRINT_PER_COUNT = 10 * 10000

		PRINT_PER_QUERY_TIME = 1 * 100
	)
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

	testParamsLength := len(testParams)
	for tps.Ops() < QUERY_NUM_LIMIT {
		param := testParams[rand.Intn(testParamsLength)]
		blocks, _ := chainInstance.GetAccountBlocksByHash(param.addr, param.origin, param.count, param.forward)
		tps.do(uint64(len(blocks)))
		tps2.doOne()
	}

	tps.Stop()
	tps.Print()

	tps2.Stop()
	tps2.Print()
}

func Benchmark_GetAccountBlockByHash(b *testing.B) {
	chainInstance := newChainInstance("insertAccountBlock", false)

	allLatestBlock, _ := chainInstance.GetAllLatestAccountBlock()

	const (
		QUERY_NUM_LIMIT        = 10000 * 10000
		BLOCK_HASH_LIST_LENGTH = 10 * 10000
		PRINT_PER_COUNT        = 1 * 10000
	)

	fmt.Printf("Create blockHashList...\n")
	var blockHashList [BLOCK_HASH_LIST_LENGTH]types.Hash
	for i := 0; i < BLOCK_HASH_LIST_LENGTH; i++ {
		block := allLatestBlock[rand.Intn(len(allLatestBlock))]
		addr := block.AccountAddress

		randomHeight := rand.Uint64() % block.Height
		if randomHeight <= 0 {
			randomHeight = 1
		}

		randomBlock, _ := chainInstance.GetAccountBlockByHeight(&addr, randomHeight)

		blockHashList[i] = randomBlock.Hash
	}

	fmt.Printf("Start query...\n")
	tps := newTps(tpsOption{
		name:          "GetAccountBlockByHash",
		printPerCount: PRINT_PER_COUNT,
	})

	tps.Start()

	for i := 0; i < QUERY_NUM_LIMIT; i++ {
		hash := blockHashList[rand.Intn(BLOCK_HASH_LIST_LENGTH)]
		_, err := chainInstance.GetAccountBlockByHash(&hash)
		if err != nil {
			b.Fatal(err)
		}
		tps.doOne()
	}
	tps.Stop()
	tps.Print()
}

type getAbByHeightParam struct {
	addr   types.Address
	height uint64
}

func Benchmark_GetAccountBlockByHeight(b *testing.B) {
	chainInstance := newChainInstance("insertAccountBlock", false)

	allLatestBlock, _ := chainInstance.GetAllLatestAccountBlock()

	const (
		QUERY_NUM_LIMIT          = 10000 * 10000
		BLOCK_HEIGHT_LIST_LENGTH = 1 * 10000
		PRINT_PER_COUNT          = 1 * 10000
	)

	fmt.Printf("Create blockHeightList...\n")
	var blockHeightList [BLOCK_HEIGHT_LIST_LENGTH]*getAbByHeightParam
	for i := 0; i < BLOCK_HEIGHT_LIST_LENGTH; i++ {
		block := allLatestBlock[rand.Intn(len(allLatestBlock))]
		randomHeight := rand.Uint64() % block.Height
		if randomHeight <= 0 {
			randomHeight = 1
		}

		blockHeightList[i] = &getAbByHeightParam{
			addr:   block.AccountAddress,
			height: randomHeight,
		}
	}

	fmt.Printf("Start query...\n")
	tps := newTps(tpsOption{
		name:          "GetAccountBlockByHeight",
		printPerCount: PRINT_PER_COUNT,
	})

	tps.Start()

	for i := 0; i < QUERY_NUM_LIMIT; i++ {
		blockHeight := blockHeightList[rand.Intn(BLOCK_HEIGHT_LIST_LENGTH)]
		_, err := chainInstance.GetAccountBlockByHeight(&blockHeight.addr, blockHeight.height)
		if err != nil {
			b.Fatal(err)
		}
		tps.doOne()
	}
	tps.Stop()
	tps.Print()
}
