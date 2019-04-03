package api

import (
	"github.com/vitelabs/go-vite/chain/unittest"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vite"
	"math/rand"
	"testing"
)

func mockVite() *vite.Vite {
	return &vite.Vite{}
}

func BenchmarkLedgerApi_GetBlockByHeight(b *testing.B) {
	b.StopTimer()
	ledgerApi := NewLedgerApi(mockVite())

	chainInstance := chain_unittest.NewChainInstance("testdata", false)
	ledgerApi.chain = chainInstance

	allLatestBlock, _ := chainInstance.GetAllLatestAccountBlock()
	allLatestBlockLength := len(allLatestBlock)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		block := allLatestBlock[rand.Intn(allLatestBlockLength)]
		addr := block.AccountAddress

		randomHeight := rand.Uint64() % block.Height
		if randomHeight <= 0 {
			randomHeight = 1
		}

		randomBlock, err := chainInstance.GetAccountBlockByHeight(&addr, randomHeight)
		if err != nil {
			b.Fatal(err)
		}

		_, err = ledgerApi.ledgerBlockToRpcBlock(randomBlock)
		if err != nil {
			b.Fatal(err)
		}
	}

	chainInstance.Destroy()
}

func BenchmarkLedgerApi_GetBlockByHash(b *testing.B) {
	b.StopTimer()

	chainInstance := chain_unittest.NewChainInstance("testdata", false)

	const (
		BLOCK_HASH_LIST_LENGTH = 10 * 10000
	)

	var blockHashList [BLOCK_HASH_LIST_LENGTH]types.Hash
	allLatestBlock, _ := chainInstance.GetAllLatestAccountBlock()

	for i := 0; i < BLOCK_HASH_LIST_LENGTH; i++ {
		block := allLatestBlock[rand.Intn(len(allLatestBlock))]
		addr := block.AccountAddress

		randomHeight := rand.Uint64() % block.Height
		if randomHeight <= 0 {
			randomHeight = 1
		}

		randomBlock, err := chainInstance.GetAccountBlockByHeight(&addr, randomHeight)
		if err != nil {
			b.Fatal(err)
		}

		blockHashList[i] = randomBlock.Hash
	}

	// destroy cache
	chainInstance.Destroy()

	chainInstance = chain_unittest.NewChainInstance("testdata", false)
	ledgerApi := NewLedgerApi(mockVite())
	ledgerApi.chain = chainInstance

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		hash := blockHashList[rand.Intn(BLOCK_HASH_LIST_LENGTH)]
		randomBlock, err := chainInstance.GetAccountBlockByHash(&hash)
		if err != nil {
			b.Fatal(err)
		}
		_, err = ledgerApi.ledgerBlockToRpcBlock(randomBlock)
		if err != nil {
			b.Fatal(err)
		}

	}

	chainInstance.Destroy()
}
