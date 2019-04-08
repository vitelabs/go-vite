package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"testing"
)

func TestChain_DeleteSnapshotBlocks(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	chainInstance, accounts, snapshotBlockList := SetUp(t, 168, 24, 2)

	for i := 0; i < 2; i++ {
		InsertAccountBlock(t, chainInstance, accounts, rand.Intn(1000), rand.Intn(12))

		testChainAll(t, chainInstance, accounts, snapshotBlockList)

		//deleteCount := (rand.Uint64() % 5) + 1
		deleteCount := uint64(2)

		DeleteSnapshotBlocks(t, chainInstance, accounts, deleteCount)
		snapshotBlockList = snapshotBlockList[:uint64(len(snapshotBlockList))-deleteCount]

		testChainAll(t, chainInstance, accounts, snapshotBlockList)
	}

	TearDown(chainInstance)
}

func DeleteSnapshotBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, count uint64) {
	snapshotBlocksToDelete, err := chainInstance.GetSnapshotBlocks(chainInstance.GetLatestSnapshotBlock().Hash, false, count)

	if err != nil {
		t.Fatal(err)
	}
	if _, err := chainInstance.DeleteSnapshotBlocksToHeight(chainInstance.GetLatestSnapshotBlock().Height + 1 - count); err != nil {
		snapshotBlocksStr := ""
		for _, snapshotBlock := range snapshotBlocksToDelete {
			snapshotBlocksStr += fmt.Sprintf("%+v, ", snapshotBlock)
		}

		t.Fatal(fmt.Sprintf("Error: %s, snapshotBlocks: %s", err, snapshotBlocksStr))
	}

	for _, account := range accounts {
		account.DeleteSnapshotBlocks(accounts, snapshotBlocksToDelete)
	}
}
