package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func TestChain_OnRoad(t *testing.T) {
	chainInstance, accounts, _, addrList, _, _ := SetUp(t, 123, 1231, 12)

	testOnRoad(t, chainInstance, accounts, addrList)

	TearDown(chainInstance)
}

func testOnRoad(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, addrList []types.Address) {
	t.Run("HasOnRoadBlocks", func(t *testing.T) {
		HasOnRoadBlocks(t, chainInstance, accounts)
	})

	t.Run("GetOnRoadBlocksHashList", func(t *testing.T) {
		GetOnRoadBlocksHashList(t, chainInstance, accounts, addrList)
	})
}

func HasOnRoadBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		addr := account.addr
		result, err := chainInstance.HasOnRoadBlocks(addr)
		if err != nil {
			t.Fatal(err)
		}
		account := accounts[addr]
		if result && len(account.UnreceivedBlocks) <= 0 {
			chainInstance.HasOnRoadBlocks(addr)
			t.Fatal(fmt.Sprintf("%s", addr))
		}

		if !result && len(account.UnreceivedBlocks) > 0 {
			t.Fatal("error")
		}
	}
}

func GetOnRoadBlocksHashList(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account, addrList []types.Address) {
	countPerPage := 10

	for _, addr := range addrList {
		pageNum := 0
		hashSet := make(map[types.Hash]struct{})
		account := accounts[addr]

		for {
			hashList, err := chainInstance.GetOnRoadBlocksHashList(addr, pageNum, 10)
			if err != nil {
				t.Fatal(err)
			}

			hashListLen := len(hashList)
			if hashListLen <= 0 {
				break
			}

			if hashListLen > countPerPage {
				t.Fatal(err)
			}

			for _, hash := range hashList {
				if _, ok := hashSet[hash]; ok {
					t.Fatal(err)
				}

				hashSet[hash] = struct{}{}
				hasUnReceive := false
				for _, unReceiveBlock := range account.UnreceivedBlocks {
					if unReceiveBlock.AccountBlock.Hash == hash {
						hasUnReceive = true
						break
					}
				}

				if !hasUnReceive {
					t.Fatal("error")
				}
			}
			pageNum++
		}

		if len(hashSet) != len(account.UnreceivedBlocks) {
			t.Fatal("error")
		}
	}
}
