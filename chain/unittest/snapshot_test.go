package chain_unittest

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func getAccountNumber(chainInstance chain.Chain) uint64 {
	blocks, _ := chainInstance.GetAllLatestAccountBlock()
	return uint64(len(blocks))

}

func Test_SnapshotStateTrie(t *testing.T) {
	chainInstance := NewChainInstance("testdata", false)

	const (
		MAX_SNAPSHOTBLOCK_COUNT = 100
	)

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	count := latestSnapshotBlock.Height
	if count > MAX_SNAPSHOTBLOCK_COUNT {
		count = MAX_SNAPSHOTBLOCK_COUNT
	}

	blocks, err := chainInstance.GetSnapshotBlocksByHash(&latestSnapshotBlock.Hash, count, false, false)
	if err != nil {
		t.Fatal(err)
	}
	accountSet := make(map[types.Address]struct{})
	for _, block := range blocks {
		trie := chainInstance.GetStateTrie(&block.StateHash)
		iterator := trie.NewIterator(nil)

		for {
			key, value, ok := iterator.Next()
			if !ok {
				break
			}

			addr, err := types.BytesToAddress(key)
			if err != nil {
				t.Fatal("key is wrong, error is " + err.Error())
			}

			accountSet[addr] = struct{}{}

			confirmedBlock, err := chainInstance.GetConfirmAccountBlock(block.Height, &addr)
			if err != nil {
				t.Fatal("GetConfirmAccountBlock failed, error is " + err.Error())
			}

			accountStateHash, err := types.BytesToHash(value)
			if err != nil {
				t.Fatal("BytesToHash failed, error is " + err.Error())
			}

			if confirmedBlock.StateHash != accountStateHash {
				t.Fatal("accountStateHash is wrong， error is " + err.Error())
			}

		}

	}

	accountSetLength := uint64(len(accountSet))
	accountNumber := getAccountNumber(chainInstance)
	if accountSetLength != accountNumber {
		t.Fatal(fmt.Sprintf("accountSetLength != accountNumber, accountSetLength is %d, accountNumber is %d", accountSetLength, accountNumber))
	}

}
