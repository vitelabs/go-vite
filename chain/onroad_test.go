package chain

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func TestChain_OnRoad(t *testing.T) {
	chainInstance, accounts, _ := SetUp(t, 123, 1231, 12)

	testOnRoad(t, chainInstance, accounts)

	TearDown(chainInstance)
}

func testOnRoad(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	t.Run("HasOnRoadBlocks", func(t *testing.T) {
		HasOnRoadBlocks(t, chainInstance, accounts)
	})

	t.Run("GetOnRoadBlocksHashList", func(t *testing.T) {
		GetOnRoadBlocksHashList(t, chainInstance, accounts)
	})
}

func HasOnRoadBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		result, err := chainInstance.HasOnRoadBlocks(addr)
		if err != nil {
			t.Fatal(err)
		}

		if result && len(account.OnRoadBlocks) <= 0 {
			t.Fatal(fmt.Sprintf("%s", addr))
		}

		if !result && len(account.OnRoadBlocks) > 0 {
			t.Fatal(fmt.Sprintf("%+v\n", account.OnRoadBlocks))
		}
	}
}

func TestGetOnRoadBlocks(t *testing.T) {
	addr, err := types.HexToAddress("vite_ee0f97c3947596401dd4ba952cd77e69c96a7109fe80924db7")
	if err != nil {
		t.Fatal(err)
	}
	chainInstance, err := NewChainInstance("unit_test", false)
	if err != nil {
		t.Fatal(err)
	}

	hashList, err := chainInstance.GetOnRoadBlocksHashList(addr, 0, 100)

	fmt.Println(hashList)

	sendHash, err := types.HexToHash("301f17c6ccc1ba9c66c677c52748893aa176514e792cdd5dfa855fc105a31ba4")
	block, err := chainInstance.GetReceiveAbBySendAb(sendHash)
	fmt.Println(block, err)

}

func GetOnRoadBlocksHashList(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	countPerPage := 10

	for addr, account := range accounts {
		pageNum := 0
		hashSet := make(map[types.Hash]struct{})

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
				t.Fatal("error")
			}

			for _, hash := range hashList {
				if _, ok := hashSet[hash]; ok {
					t.Fatal(fmt.Sprintf("Hash set is %+v, hashList is %+v", hashSet, hashList))
				}

				hashSet[hash] = struct{}{}

				if _, hasUnReceive := account.OnRoadBlocks[hash]; !hasUnReceive {
					key := chain_utils.CreateOnRoadPrefixKey(&addr)

					iter := chainInstance.indexDB.Store().NewIterator(util.BytesPrefix(key))
					defer iter.Release()

					startIndex := pageNum * countPerPage
					endIndex := (pageNum + 1) * countPerPage

					index := 0
					for iter.Next() && index < endIndex {

						if index >= startIndex {
							hash, err := types.BytesToHash(iter.Value())
							if err != nil {
								t.Fatal(err)
							}

							fmt.Printf("onroad list: key: %d value: %s\n", iter.Key(), hash)
						}
						index++
					}

					if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
						t.Fatal(err)
					}

					fmt.Printf("Hash is %s, addr: %s, hashList is %+v，account.OnRoadBlocks: %+v\n", hash, addr, hashList, account.OnRoadBlocks)
					panic(fmt.Sprintf("Hash is %s, addr: %s, hashList is %+v，account.OnRoadBlocks: %+v\n", hash, addr, hashList, account.OnRoadBlocks))

				}
			}
			pageNum++
		}

		if len(hashSet) != len(account.OnRoadBlocks) {
			onRoadBlocks := make(map[types.Hash]struct{})
			for hash := range account.OnRoadBlocks {
				onRoadBlocks[hash] = struct{}{}
			}
			for hash := range onRoadBlocks {
				if _, ok := hashSet[hash]; ok {
					delete(onRoadBlocks, hash)
				}
			}
			t.Fatal(fmt.Sprintf("addr %s, lack account.OnRoadBlocks: %+v", account.Addr, onRoadBlocks))
		}
	}
}
