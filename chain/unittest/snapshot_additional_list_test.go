package chain_unittest

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"math/rand"
	"testing"
)

func Test_SaList_Add(t *testing.T) {
	const (
		MIN_BLOCKS_PERSNAPSOHT     = 10
		MAX_BLOCKS_PERSNAPSOHT     = 20
		SNAPSHOT_BLOCKS            = 60 * 60 * 24 * 2
		TEST_SNAPSHOT_BLOCKS_RANGE = 60 * 60 * 24

		TEST_TIMES = 10
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

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	snapshotBlockCount := uint64(0)
	if latestSnapshotBlock.Height < SNAPSHOT_BLOCKS {
		snapshotBlockCount = SNAPSHOT_BLOCKS - latestSnapshotBlock.Height
	}
	sbList := make([]*ledger.SnapshotBlock, snapshotBlockCount)
	for sbIndex := uint64(0); sbIndex < snapshotBlockCount; sbIndex++ {
		txNums := rand.Intn(MAX_BLOCKS_PERSNAPSOHT-MIN_BLOCKS_PERSNAPSOHT) + MIN_BLOCKS_PERSNAPSOHT

		for txIndex := 0; txIndex < txNums; txIndex++ {
			randomAccount := accounts[rand.Intn(accountLen)]
			var blocks []*vm_context.VmAccountBlock
			if randomAccount.HasUnreceivedBlock() {
				blocks = randomAccount.CreateResponseTx(cTxOptions)
			} else {
				toAccount := accounts[rand.Intn(accountLen)]
				blocks = randomAccount.CreateRequestTx(toAccount, cTxOptions)
			}
			if err := chainInstance.InsertAccountBlocks(blocks); err != nil {
				t.Fatal(err)
			}
		}

		sb := test_tools.CreateSnapshotBlock(chainInstance, &test_tools.SnapshotOptions{
			MockTrie: false,
		})
		chainInstance.InsertSnapshotBlock(sb)
		sbList[sbIndex] = sb

		if (sbIndex+1)%100 == 0 {
			fmt.Printf("Insert %d snapshot blocks\n", sbIndex+1)
		}
	}

	sbListLength := len(sbList)
	fmt.Printf("Insert %d snapshot blocks in total\n", sbListLength)

	latestSnapshotBlock = chainInstance.GetLatestSnapshotBlock()
	randomStartHeight := latestSnapshotBlock.Height - TEST_SNAPSHOT_BLOCKS_RANGE + 1
	randomEndHeight := latestSnapshotBlock.Height

	for i := 0; i < TEST_TIMES; i++ {
		randomHeight := rand.Uint64()%(randomEndHeight-randomStartHeight) + randomStartHeight

		randomSnapshotBlock, err := chainInstance.GetSnapshotBlockByHeight(randomHeight)
		if err != nil {
			t.Fatal(err)
		}

		aggregateQuota, err := chainInstance.SaList().GetAggregateQuota(randomSnapshotBlock)
		if err != nil {
			t.Fatal(err)
		}

		sbs, err := chainInstance.GetSnapshotBlocksByHeight(randomSnapshotBlock.Height, uint64(60*60), false, true)
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
			err := errors.New(fmt.Sprintf("aggregateQuota != subLedgerTotalQuota, aggregateQuota is %d, subLedgerTotalQuota is %d\n", aggregateQuota, subLedgerTotalQuota))
			t.Fatal(err)
		}

		fmt.Printf("Test %d: aggregateQuota is %d,  subLedgerTotalQuota is %d\n", i+1, aggregateQuota, subLedgerTotalQuota)
	}

}
