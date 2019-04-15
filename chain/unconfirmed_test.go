package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func TestUnconfirmed(t *testing.T) {
	chainInstance, accounts, _ := SetUp(t, 2, 30, 100)

	testUnconfirmed(t, chainInstance, accounts)
	TearDown(chainInstance)
}

func testUnconfirmed(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	t.Run("GetUnconfirmedBlocks", func(t *testing.T) {
		GetUnconfirmedBlocks(t, chainInstance, accounts)
	})
}

func GetUnconfirmedBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	queryUnconfirmedBlocks := chainInstance.cache.GetUnconfirmedBlocks()

	// query all unconfirmed blocks
	for _, block := range queryUnconfirmedBlocks {
		account := accounts[block.AccountAddress]
		if _, ok := account.UnconfirmedBlocks[block.Hash]; !ok {
			t.Fatal("error")
		}
	}

	count := 0
	for addr, account := range accounts {
		count += len(account.UnconfirmedBlocks)

		queryBlocks := chainInstance.GetUnconfirmedBlocks(addr)
		if len(queryBlocks) != len(account.UnconfirmedBlocks) {
			t.Fatal(fmt.Sprintf("snapshot: %d, addr: %s, queryBlocks: %+v \n account.UnconfirmedBlocks: %+v",
				chainInstance.GetLatestSnapshotBlock().Height, addr, queryBlocks, account.UnconfirmedBlocks))
		}

		for _, queryBlock := range queryBlocks {
			if _, ok := account.UnconfirmedBlocks[queryBlock.Hash]; !ok {
				t.Fatal("error")
			}
		}
	}

	if len(queryUnconfirmedBlocks) != count {
		t.Fatal("error")
	}

}
