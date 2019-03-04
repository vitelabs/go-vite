package chain_unittest

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
)

func Test_ReceiveHeights(t *testing.T) {
	const PRINT_PER_ACCOUNTS = 1000
	chainInstance := NewChainInstance("testdata", false)

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	allLatestAccountBlock, err := chainInstance.GetAllLatestAccountBlock()
	if err != nil {
		panic(err)
	}
	//allAccounts := make([]*ledger.Account, 0, len(allLatestAccountBlock))

	allOnRoadBlocks := make(map[types.Address][]*ledger.AccountBlock)

	accountNum := 0
	for _, latestAccountBlock := range allLatestAccountBlock {
		accountNum++
		addr := latestAccountBlock.AccountAddress
		//account, err := chainInstance.GetAccount(&addr)
		//if err != nil {
		//	panic(err)
		//}
		//
		//allAccounts = append(allAccounts, account)
		if onRoadBlocks, err := chainInstance.GetOnRoadBlocksBySendAccount(&addr, latestSnapshotBlock.Height); err != nil {
			panic(err)
		} else if len(onRoadBlocks) > 0 {
			allOnRoadBlocks[addr] = onRoadBlocks
		}

		if accountNum%PRINT_PER_ACCOUNTS == 0 || accountNum == len(allLatestAccountBlock) {
			fmt.Printf("Has query %d accounts(total %d accounts)\n", accountNum, len(allLatestAccountBlock))
		}
	}

	fmt.Printf("======Test result======")
	for addr, blocks := range allOnRoadBlocks {
		fmt.Printf("%s: %d\n", addr, len(blocks))
	}
	fmt.Printf("======Test result======")

}
