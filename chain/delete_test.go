package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"log"
	"net/http"
	_ "net/http/pprof"
	"testing"
)

func TestChain_DeleteSnapshotBlocks(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	chainInstance, accounts, snapshotBlockList := SetUp(t, 5, 24, 2)
	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	deleteCount := uint64(3)
	DeleteSnapshotBlocks(t, chainInstance, accounts, deleteCount)
	snapshotBlockList = snapshotBlockList[:uint64(len(snapshotBlockList))-deleteCount]

	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	TearDown(chainInstance)
}

func DeleteSnapshotBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, count uint64) {
	snapshotBlocksToDelete, err := chainInstance.GetSnapshotBlocks(chainInstance.GetLatestSnapshotBlock().Hash, false, count)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chainInstance.DeleteSnapshotBlocksToHeight(chainInstance.GetLatestSnapshotBlock().Height + 1 - count); err != nil {
		t.Fatal(err)
	}

	for _, account := range accounts {
		account.DeleteSnapshotBlocks(accounts, snapshotBlocksToDelete)
	}
}
