package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func TestChain_OnRoad(t *testing.T) {
	chainInstance, accounts, _ := SetUp(123, 1231, 12)

	testOnRoad(t, chainInstance, accounts)

	TearDown(chainInstance)
}

func testOnRoad(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	//t.Run("HasOnRoadBlocks", func(t *testing.T) {
	//	HasOnRoadBlocks(chainInstance, accounts)
	//})
	//
	t.Run("GetOnRoadBlocksHashList", func(t *testing.T) {
		for addr, _ := range accounts {
			fmt.Println(chainInstance.GetOnRoadBlocksByAddr(addr, 0, 10))
		}
	})
}

func testOnRoadNoTesting(chainInstance *chain, accounts map[types.Address]*Account) {

	//HasOnRoadBlocks(chainInstance, accounts)
	//
	//GetOnRoadBlocksHashList(chainInstance, accounts)

}

//func HasOnRoadBlocks(chainInstance *chain, accounts map[types.Address]*Account) {
//	for addr, account := range accounts {
//		result, err := chainInstance.HasOnRoadBlocks(addr)
//		if err != nil {
//			panic(err)
//		}
//
//		if result && len(account.OnRoadBlocks) <= 0 {
//			panic(fmt.Sprintf("%s", addr))
//		}
//
//		if !result && len(account.OnRoadBlocks) > 0 {
//			panic(fmt.Sprintf("%+v\n", account.OnRoadBlocks))
//		}
//	}
//}
//
//func GetOnRoadBlocksHashList(chainInstance *chain, accounts map[types.Address]*Account) {
//	countPerPage := 10
//
//	for addr, account := range accounts {
//		pageNum := 0
//		hashSet := make(map[types.Hash]struct{})
//
//		for {
//			hashList, err := chainInstance.GetOnRoadBlocksByAddr(addr, pageNum, 10)
//			if err != nil {
//				panic(err)
//			}
//
//			hashListLen := len(hashList)
//			if hashListLen <= 0 {
//				break
//			}
//
//			if hashListLen > countPerPage {
//				panic("error")
//			}
//
//			for _, hash := range hashList {
//				if _, ok := hashSet[hash]; ok {
//					panic(fmt.Sprintf("Hash set is %+v, hashList is %+v", hashSet, hashList))
//				}
//
//				hashSet[hash] = struct{}{}
//
//				if _, hasUnReceive := account.OnRoadBlocks[hash]; !hasUnReceive {
//					key := chain_utils.CreateOnRoadPrefixKey(&addr)
//
//					iter := chainInstance.indexDB.Store().NewIterator(util.BytesPrefix(key))
//					defer iter.Release()
//
//					startIndex := pageNum * countPerPage
//					endIndex := (pageNum + 1) * countPerPage
//
//					index := 0
//					for iter.Next() && index < endIndex {
//
//						if index >= startIndex {
//							hash, err := types.BytesToHash(iter.Value())
//							if err != nil {
//								panic(err)
//							}
//
//							fmt.Printf("onroad list: key: %d value: %s\n", iter.Key(), hash)
//						}
//						index++
//					}
//
//					if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
//						panic(err)
//					}
//
//					fmt.Printf("Hash is %s, addr: %s, hashList is %+v，account.OnRoadBlocks: %+v\n", hash, addr, hashList, account.OnRoadBlocks)
//					panic(fmt.Sprintf("Hash is %s, addr: %s, hashList is %+v，account.OnRoadBlocks: %+v\n", hash, addr, hashList, account.OnRoadBlocks))
//
//				}
//			}
//			pageNum++
//		}
//
//		if len(hashSet) != len(account.OnRoadBlocks) {
//			onRoadBlocks := make(map[types.Hash]struct{})
//			for hash := range account.OnRoadBlocks {
//				onRoadBlocks[hash] = struct{}{}
//			}
//			for hash := range onRoadBlocks {
//				if _, ok := hashSet[hash]; ok {
//					delete(onRoadBlocks, hash)
//				}
//			}
//			panic(fmt.Sprintf("addr %s, lack account.OnRoadBlocks: %+v", account.Addr, onRoadBlocks))
//		}
//	}
//}
