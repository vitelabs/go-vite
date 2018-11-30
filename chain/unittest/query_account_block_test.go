package chain_unittest

import (
	"fmt"
	"testing"
)

func Test_GetAccountBlocksByHash(t *testing.T) {
	chainInstance := newChainInstance("testdata", false, false)
	allLatestBlock, _ := chainInstance.GetAllLatestAccountBlock()
	fmt.Printf("allLatestBlock length: %d\n", len(allLatestBlock))
	for _, block := range allLatestBlock {
		addr := block.AccountAddress
		blocksFalse, err := chainInstance.GetAccountBlocksByHash(addr, &block.Hash, block.Height, false)
		if err != nil {
			t.Fatal(err)
		}
		if uint64(len(blocksFalse)) != block.Height {
			t.Fatal("length is error!")
		}

		blocksTrue, err := chainInstance.GetAccountBlocksByHash(addr, nil, block.Height, true)
		if err != nil {
			t.Fatal(err)
		}
		if uint64(len(blocksTrue)) != block.Height {
			t.Fatal("length is error!")
		}

	}
}
