package chain_unittest

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
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
	chainInstance := newChainInstance("unit_test", false, true)

	accounts := test_tools.MakeAccounts(10, chainInstance)

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	snapshotBlockCount := uint64(0)
	if latestSnapshotBlock.Height < SNAPSHOT_BLOCKS {
		snapshotBlockCount = SNAPSHOT_BLOCKS - latestSnapshotBlock.Height
	}
	if err := makeSnapshotBlocks(chainInstance, accounts, snapshotBlockCount, MIN_BLOCKS_PERSNAPSOHT, MAX_BLOCKS_PERSNAPSOHT); err != nil {
		t.Fatal(err)
	}

	// TEST normal
	latestSnapshotBlock = chainInstance.GetLatestSnapshotBlock()
	randomStartHeight := latestSnapshotBlock.Height - TEST_SNAPSHOT_BLOCKS_RANGE + 1
	randomEndHeight := latestSnapshotBlock.Height

	fmt.Printf("[TEST normal]\n")
	for i := 0; i < TEST_TIMES; i++ {
		randomHeight := rand.Uint64()%(randomEndHeight-randomStartHeight) + randomStartHeight

		randomSnapshotBlock, err := chainInstance.GetSnapshotBlockByHeight(randomHeight)
		if err != nil {
			t.Fatal(err)
		}

		result := checkTotalQuota(t, randomSnapshotBlock, chainInstance)
		fmt.Printf("%d. %s", i+1, result)
	}

	// TEST normal 2
	testEndHeight := latestSnapshotBlock.Height
	testStartHeight := testEndHeight - 100 + 1
	fmt.Printf("[TEST normal2]\n")
	for h := testStartHeight; h <= testEndHeight; h++ {
		randomSnapshotBlock, err := chainInstance.GetSnapshotBlockByHeight(h)
		if err != nil {
			t.Fatal(err)
		}

		result := checkTotalQuota(t, randomSnapshotBlock, chainInstance)
		fmt.Printf("height %d. %s", h, result)
	}

	// TEST delete
	targetHeight := latestSnapshotBlock.Height - 10 + 1
	sbList, _, err := chainInstance.DeleteSnapshotBlocksToHeight(targetHeight)
	if err != nil {
		t.Fatal(err)
	}

	for _, sb := range sbList {
		_, err := chainInstance.SaList().GetAggregateQuota(sb)
		if err == nil {
			t.Fatal(err)
		}
	}

	testEndHeight = targetHeight - 1
	testStartHeight = testEndHeight - TEST_SNAPSHOT_BLOCKS_RANGE + 1

	fmt.Printf("[TEST delete]\n")
	for i := 0; i < TEST_TIMES; i++ {
		randomHeight := rand.Uint64()%(testEndHeight-testStartHeight) + testStartHeight

		randomSnapshotBlock, err := chainInstance.GetSnapshotBlockByHeight(randomHeight)
		if err != nil {
			t.Fatal(err)
		}

		result := checkTotalQuota(t, randomSnapshotBlock, chainInstance)
		fmt.Printf("%d. %s", i+1, result)
	}

	// TEST add
	if err := makeSnapshotBlocks(chainInstance, accounts, 5, MIN_BLOCKS_PERSNAPSOHT, MAX_BLOCKS_PERSNAPSOHT); err != nil {
		t.Fatal(err)
	}
	latestSnapshotBlock = chainInstance.GetLatestSnapshotBlock()

	testEndHeight = latestSnapshotBlock.Height
	testStartHeight = testEndHeight - 5 + 1

	fmt.Printf("[TEST add after delete]\n")
	for h := testStartHeight; h <= testEndHeight; h++ {
		randomSnapshotBlock, err := chainInstance.GetSnapshotBlockByHeight(h)
		if err != nil {
			t.Fatal(err)
		}

		result := checkTotalQuota(t, randomSnapshotBlock, chainInstance)
		fmt.Printf("height %d: %s", h, result)
	}
}

func makeSnapshotBlocks(chainInstance chain.Chain, accounts []*test_tools.Account, num uint64, minAccountBlocks, maxAccountBlocks int) error {
	accountLen := len(accounts)

	quota := uint64(2)
	cTxOptions := &test_tools.CreateTxOptions{
		MockVmContext: true,
		MockSignature: true,
		Quota:         quota,
	}

	for sbIndex := uint64(0); sbIndex < num; sbIndex++ {
		txNums := rand.Intn(maxAccountBlocks-minAccountBlocks) + minAccountBlocks

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
				return err
			}
		}

		sb := test_tools.CreateSnapshotBlock(chainInstance, &test_tools.SnapshotOptions{
			MockTrie: false,
		})
		chainInstance.InsertSnapshotBlock(sb)

		if (sbIndex+1)%100 == 0 {
			fmt.Printf("Insert %d snapshot blocks\n", sbIndex+1)
		}
	}

	fmt.Printf("Insert %d snapshot blocks in total\n", num)
	return nil
}
func checkTotalQuota(t *testing.T, snapshotBlock *ledger.SnapshotBlock, chainInstance chain.Chain) string {
	aggregateQuota, err := chainInstance.SaList().GetAggregateQuota(snapshotBlock)
	if err != nil {
		t.Fatal(err)
	}

	sbs, err := chainInstance.GetSnapshotBlocksByHeight(snapshotBlock.Height, uint64(60*60), false, true)
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

	return fmt.Sprintf("Test result: aggregateQuota is %d,  subLedgerTotalQuota is %d\n", aggregateQuota, subLedgerTotalQuota)
}
