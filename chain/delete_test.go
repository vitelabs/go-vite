package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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
	for i := 0; i < 1; i++ {
		chainInstance, accounts, snapshotBlockList := SetUp(t, 100, 960, 2)

		snapshotBlockList = testInsertAndDelete(t, chainInstance, accounts, snapshotBlockList)

		TearDown(chainInstance)
	}

}

func testInsertAndDelete(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {

	for i := 0; i < 5; i++ {
		t.Run("deleteSnapshotBlocks", func(t *testing.T) {
			snapshotBlockList = testDeleteSnapshotBlocks(t, chainInstance, accounts, snapshotBlockList, rand.Intn(8))
		})

		t.Run("deleteAccountBlocks", func(t *testing.T) {
			snapshotBlockList = testDeleteAccountBlocks(t, chainInstance, accounts, snapshotBlockList)
		})

		t.Run("deleteAccountBlocks", func(t *testing.T) {
			snapshotBlockList = testDeleteAccountBlocks(t, chainInstance, accounts, snapshotBlockList)
		})

	}

	//t.Run("DeleteMany", func(t *testing.T) {
	//	snapshotBlockList = testDeleteMany(t, chainInstance, accounts, snapshotBlockList)
	//})

	//t.Run("deleteSnapshotBlocks", func(t *testing.T) {
	//	snapshotBlockList = testDeleteSnapshotBlocks(t, chainInstance, accounts, snapshotBlockList, rand.Intn(10))
	//})
	//
	//t.Run("deleteAccountBlocks", func(t *testing.T) {
	//	snapshotBlockList = testDeleteAccountBlocks(t, chainInstance, accounts, snapshotBlockList)
	//})

	return snapshotBlockList
}

func testDeleteMany(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {
	snapshotBlockList = append(snapshotBlockList, InsertAccountBlock(t, chainInstance, accounts, 15000, 3, false)...)
	deleteCount := 3500
	deleteSnapshotBlocks(t, chainInstance, accounts, uint64(deleteCount))

	snapshotBlockList = snapshotBlockList[:len(snapshotBlockList)-deleteCount]

	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	return snapshotBlockList

}

func testDeleteSnapshotBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock, deleteCount int) []*ledger.SnapshotBlock {
	insertCount := rand.Intn(100)
	snapshotPerNum := rand.Intn(5)

	snapshotBlockList = append(snapshotBlockList, InsertAccountBlock(t, chainInstance, accounts, insertCount, snapshotPerNum, false)...)

	if deleteCount > len(snapshotBlockList) {
		lackNum := deleteCount - len(snapshotBlockList) + 10
		snapshotBlockList = append(snapshotBlockList, InsertAccountBlock(t, chainInstance, accounts, lackNum*20, 20, false)...)
	}

	//testChainAll(t, chainInstance, accounts, snapshotBlockList)
	//GetUnconfirmedBlocks(t, chainInstanc	e, accounts)
	//GetOnRoadBlocksHashList(t, chainInstance, accounts)
	GetLatestAccountBlock(t, chainInstance, accounts)
	//GetStorageIterator(t, chainInstance, accounts)
	//GetBalance(t, chainInstance, accounts)
	//
	//GetConfirmedBalanceList(t, chainInstance, accounts, snapshotBlockList)

	//NewStorageDatabase(t, chainInstance, accounts, snapshotBlockList)

	deleteSnapshotBlocks(t, chainInstance, accounts, uint64(deleteCount))
	//
	snapshotBlockList = snapshotBlockList[:len(snapshotBlockList)-deleteCount]

	//NewStorageDatabase(t, chainInstance, accounts, snapshotBlockList)

	//testChainAll(t, chainInstance, accounts, snapshotBlockList)
	//GetUnconfirmedBlocks(t, chainInstance, accounts)
	//GetOnRoadBlocksHashList(t, chainInstance, accounts)
	GetLatestAccountBlock(t, chainInstance, accounts)
	//GetStorageIterator(t, chainInstance, accounts)
	//GetBalance(t, chainInstance, accounts)

	//GetConfirmedBalanceList(t, chainInstance, accounts, snapshotBlockList)

	return snapshotBlockList
}

func testDeleteAccountBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {

	snapshotBlockList = append(snapshotBlockList, InsertAccountBlock(t, chainInstance, accounts, rand.Intn(100), 5, false)...)
	//testChainAll(t, chainInstance, accounts, snapshotBlockList)
	//GetOnRoadBlocksHashList(t, chainInstance, accounts)
	//testChainAll(t, chainInstance, accounts, snapshotBlockList)

	//GetUnconfirmedBlocks(t, chainInstance, accounts)
	//GetOnRoadBlocksHashList(t, chainInstance, accounts)
	GetLatestAccountBlock(t, chainInstance, accounts)
	//GetStorageIterator(t, chainInstance, accounts)
	//GetBalance(t, chainInstance, accounts)
	//GetConfirmedBalanceList(t, chainInstance, accounts, snapshotBlockList)

	//GetStorageIterator(t, chainInstance, accounts)
	//NewStorageDatabase(t, chainInstance, accounts, snapshotBlockList)

	deleteAccountBlocks(t, chainInstance, accounts)

	//NewStorageDatabase(t, chainInstance, accounts, snapshotBlockList)
	//testChainAll(t, chainInstance, accounts, snapshotBlockList)

	//testChainAll(t, chainInstance, accounts, snapshotBlockList)
	//GetOnRoadBlocksHashList(t, chainInstance, accounts)
	//GetUnconfirmedBlocks(t, chainInstance, accounts)
	//GetOnRoadBlocksHashList(t, chainInstance, accounts)
	GetLatestAccountBlock(t, chainInstance, accounts)
	//GetStorageIterator(t, chainInstance, accounts)
	//GetBalance(t, chainInstance, accounts)

	//GetStorageIterator(t, chainInstance, accounts)

	//GetConfirmedBalanceList(t, chainInstance, accounts, snapshotBlockList)

	return snapshotBlockList
}

func deleteSnapshotBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, count uint64) {
	if count <= 0 {
		return
	}
	snapshotBlocksToDelete, err := chainInstance.GetSnapshotBlocks(chainInstance.GetLatestSnapshotBlock().Hash, false, count)

	if err != nil {
		t.Fatal(err)
	}

	hasStorageRedoLog, err := chainInstance.stateDB.StorageRedo().HasRedo(snapshotBlocksToDelete[len(snapshotBlocksToDelete)-1].Height)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := chainInstance.DeleteSnapshotBlocksToHeight(snapshotBlocksToDelete[len(snapshotBlocksToDelete)-1].Height); err != nil {
		snapshotBlocksStr := ""
		for _, snapshotBlock := range snapshotBlocksToDelete {
			snapshotBlocksStr += fmt.Sprintf("%+v, ", snapshotBlock)
		}

		t.Fatal(fmt.Sprintf("Error: %s, snapshotBlocks: %s", err, snapshotBlocksStr))
	}

	for _, account := range accounts {
		account.DeleteSnapshotBlocks(accounts, snapshotBlocksToDelete, hasStorageRedoLog)
	}
}

func deleteAccountBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	unconfirmedBlocks := chainInstance.cache.GetUnconfirmedBlocks()
	if len(unconfirmedBlocks) <= 0 {
		return
	}

	unconfirmedBlock := unconfirmedBlocks[rand.Intn(len(unconfirmedBlocks))]

	account := accounts[unconfirmedBlock.AccountAddress]
	deletedAccountBlocks, err := chainInstance.DeleteAccountBlocks(account.Addr, unconfirmedBlock.Hash)
	if err != nil {
		t.Fatal(err)
	}

	for i := len(deletedAccountBlocks) - 1; i >= 0; i-- {
		ab := deletedAccountBlocks[i]
		fmt.Printf("test delete by ab %s %d %s\n", ab.AccountAddress, ab.Height, ab.Hash)

		accounts[ab.AccountAddress].deleteAccountBlock(accounts, deletedAccountBlocks[i].Hash)
		accounts[ab.AccountAddress].rollbackLatestBlock()

	}
	//
	//onRoadBlocksCache := make(map[types.Hash]struct{})
	//
	//deleteMemAccountBlock(accounts, account, unconfirmedBlock, onRoadBlocksCache)
	//
	//for fromBlockHash := range onRoadBlocksCache {
	//	var onRoadSendBlock *ledger.AccountBlock
	//	for _, account := range accounts {
	//		if len(account.SendBlocksMap) <= 0 {
	//			continue
	//		}
	//		if sendBlock, ok := account.SendBlocksMap[fromBlockHash]; ok {
	//			onRoadSendBlock = sendBlock
	//			break
	//		}
	//
	//	}
	//	if onRoadSendBlock == nil {
	//		continue
	//	}
	//
	//	toAccount := accounts[onRoadSendBlock.ToAddress]
	//	toAccount.AddOnRoadBlock(onRoadSendBlock)
	//
	//}
}

func deleteMemAccountBlock(accounts map[types.Address]*Account, account *Account,
	toBlock *ledger.AccountBlock, onRoadBlocksCache map[types.Hash]struct{}) {

	deleteSendBlock := func(sendBlock *ledger.AccountBlock) {
		delete(onRoadBlocksCache, sendBlock.Hash)
		toAccount := accounts[sendBlock.ToAddress]
		var blockNeedDelete *ledger.AccountBlock
		for _, block := range toAccount.ReceiveBlocksMap {
			if block.FromBlockHash == sendBlock.Hash {
				blockNeedDelete = block
			}
		}
		if blockNeedDelete != nil {
			deleteMemAccountBlock(accounts, toAccount, blockNeedDelete, onRoadBlocksCache)
		} else {
			toAccount.DeleteOnRoad(sendBlock.Hash)
		}
	}

	for {
		blockToDelete := account.LatestBlock
		if blockToDelete == nil || blockToDelete.Height < toBlock.Height {
			break
		}

		if blockToDelete.IsSendBlock() {
			deleteSendBlock(blockToDelete)
		} else {
			onRoadBlocksCache[blockToDelete.FromBlockHash] = struct{}{}
		}
		for _, sendBlock := range blockToDelete.SendBlockList {
			deleteSendBlock(sendBlock)
		}

		fmt.Println("MEM DELETE ACCOUNT BLOCK", account.Addr, blockToDelete.Height, blockToDelete.Hash)

		account.deleteAccountBlock(accounts, blockToDelete.Hash)
		account.rollbackLatestBlock()

		if blockToDelete.Height <= toBlock.Height {
			break
		}
	}

}
