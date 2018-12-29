package chain_unittest

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/ledger"
	"math/rand"
	"testing"
)

func Test_SaList_Add(t *testing.T) {
	const (
		MIN_BLOCKS_PERSNAPSOHT     = 100
		MAX_BLOCKS_PERSNAPSOHT     = 200
		SNAPSHOT_BLOCKS            = 60 * 60 * 24 * 2
		TEST_SNAPSHOT_BLOCKS_RANGE = 60 * 60 * 24

		TEST_TIMES = 3
	)
	chainInstance := newChainInstance("unittest", false, true)

	accounts := test_tools.MakeAccounts(10, chainInstance)
	accountLen := len(accounts)

	quota := uint64(2)
	cTxOptions := &test_tools.CreateTxOptions{
		MockVmContext: true,
		MockSignature: true,
		Quota:         quota,
	}

	sbList := make([]*ledger.SnapshotBlock, SNAPSHOT_BLOCKS)
	for sbIndex := 0; sbIndex < SNAPSHOT_BLOCKS; sbIndex++ {
		txNums := rand.Intn(MAX_BLOCKS_PERSNAPSOHT-MIN_BLOCKS_PERSNAPSOHT) + MIN_BLOCKS_PERSNAPSOHT

		for txIndex := 0; txIndex < txNums; txIndex++ {
			randomAccount := accounts[rand.Intn(accountLen)]
			if randomAccount.HasUnreceivedBlock() {
				randomAccount.CreateResponseTx(cTxOptions)
			} else {
				toAccount := accounts[rand.Intn(accountLen)]
				randomAccount.CreateRequestTx(toAccount, cTxOptions)
			}
		}

		sb := test_tools.CreateSnapshotBlock(chainInstance, &test_tools.SnapshotOptions{
			MockTrie: false,
		})
		chainInstance.InsertSnapshotBlock(sb)
		sbList[sbIndex] = sb

		if sbIndex%99 == 0 {
			fmt.Printf("Insert %d snapshot blocks\n", sbIndex+1)
		}
	}

	sbListLength := len(sbList)
	fmt.Printf("There are %d snapshot blocks\n", sbListLength)

	randomStartIndex := sbListLength - TEST_SNAPSHOT_BLOCKS_RANGE
	randomEndIndex := sbListLength - 1

	for i := 0; i < TEST_TIMES; i++ {
		randomIndex := rand.Intn(randomEndIndex-randomStartIndex) + randomStartIndex
		randomSnapshotBlock := sbList[randomIndex]

		aggregateQuota, err := chainInstance.SaList().GetAggregateQuota(randomSnapshotBlock)
		if err != nil {
			t.Fatal(err)
		}

		sbs, err := chainInstance.GetSnapshotBlocksByHeight(randomSnapshotBlock.Height, uint64(60*60), false, false)
		if err != nil {
			t.Fatal(err)
		}

		subLedger, err := chainInstance.GetConfirmSubLedgerBySnapshotBlocks(sbs)
		if err != nil {
			t.Fatal(err)
		}

		subLedgerTotalQuota := uint64(0)
		for _, blocks := range subLedger {
			for _, block := range blocks {
				subLedgerTotalQuota += block.Quota
			}
		}

		if aggregateQuota != subLedgerTotalQuota {
			err := errors.New(fmt.Sprintf("aggregateQuota != subLedgerTotalQuota, aggregateQuota is %d, subLedgerTotalQuota is %d", aggregateQuota, subLedgerTotalQuota))
			t.Fatal(err)
		}

		fmt.Printf("Test %d: aggregateQuota is %d,  subLedgerTotalQuota is %d\n", i+1, aggregateQuota, subLedgerTotalQuota)
	}

}
